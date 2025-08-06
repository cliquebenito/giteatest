// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package ssh

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/gliderlabs/ssh"
	"github.com/kballard/go-shellquote"
	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/v16/streamio"
	gossh "golang.org/x/crypto/ssh"

	asymkey_model "code.gitea.io/gitea/models/asymkey"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/git/utils"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/process"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
)

type contextKey string

const scKeyID = contextKey("sc-key-id")
const scKeyOwnerID = contextKey("sc-key-owner-id")

func getExitStatusFromError(err error) int {
	if err == nil {
		return 0
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return 1
	}

	waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		// This is a fallback and should at least let us return something useful
		// when running on Windows, even if it isn't completely accurate.
		if exitErr.Success() {
			return 0
		}

		return 1
	}

	return waitStatus.ExitStatus()
}

func sessionHandler(session ssh.Session) {
	keyID := fmt.Sprintf("%d", session.Context().Value(scKeyID).(int64))
	keyOwnerID := fmt.Sprintf("%d", session.Context().Value(scKeyOwnerID).(int64))

	command := session.RawCommand()

	log.Trace("SSH: Payload: %v", command)

	args := []string{"serv", "key-" + keyID, "--config=" + setting.CustomConf}
	log.Trace("SSH: Arguments: %v", args)

	ctx, cancel := context.WithCancel(session.Context())
	defer cancel()

	gitProtocol := ""
	for _, env := range session.Environ() {
		if strings.HasPrefix(env, "GIT_PROTOCOL=") {
			_, gitProtocol, _ = strings.Cut(env, "=")
			break
		}
	}

	cmd := exec.CommandContext(ctx, setting.AppPath, args...)
	cmd.Env = append(
		os.Environ(),
		"SSH_ORIGINAL_COMMAND="+command,
		"SKIP_MINWINSVC=1",
		"GIT_PROTOCOL="+gitProtocol,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error("SSH: StdoutPipe: %v", err)
		return
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Error("SSH: StderrPipe: %v", err)
		return
	}
	defer stderr.Close()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Error("SSH: StdinPipe: %v", err)
		return
	}
	defer stdin.Close()

	process.SetSysProcAttribute(cmd)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	if err = cmd.Start(); err != nil {
		log.Error("SSH: Start: %v", err)
		return
	}

	go func() {
		defer wg.Done()
		if _, err := io.Copy(session, stdout); err != nil {
			log.Error("Failed to write stdout to session. %s", err)
		}
	}()

	go func() {
		defer wg.Done()
		defer stderr.Close()
		if _, err := io.Copy(session.Stderr(), stderr); err != nil {
			log.Error("Failed to write stderr to session. %s", err)
		}
	}()

	// Ensure all the output has been written before we wait on the command
	// to exit.
	wg.Wait()

	// Wait for the command to exit and log any errors we get
	err = cmd.Wait()
	if err != nil {
		// Cannot use errors.Is here because ExitError doesn't implement Is
		// Thus errors.Is will do equality test NOT type comparison
		if _, ok := err.(*exec.ExitError); !ok {
			log.Error("SSH: Wait: %v", err)
		}
		if err := session.Exit(getExitStatusFromError(err)); err != nil {
			log.Error("Session failed to exit. %s", err)
		}
		return
	}

	words, err := shellquote.Split(command)
	if err != nil {
		_, _ = stderr.Read([]byte("Failed to split command"))
		return
	}
	repoPath := words[1]
	if repoPath[0] == '/' {
		repoPath = repoPath[1:]
	}
	repoPath = strings.ToLower(strings.TrimSpace(repoPath))

	rr := strings.SplitN(repoPath, "/", 2)
	if len(rr) != 2 {
		_, _ = stderr.Read([]byte("Failed to repo path"))
		return
	}

	const gitSuffix = ".git"

	if !strings.HasSuffix(repoPath, gitSuffix) {
		repoPath += gitSuffix
	}

	username := strings.ToLower(rr[0])
	reponame := strings.ToLower(strings.TrimSuffix(rr[1], gitSuffix))

	repository, err := git.OpenRepository(ctx, username, reponame, setting.RepoRootPath+"/"+repoPath)
	if err != nil {
		_, _ = stderr.Read([]byte("Failed to open repository"))
		return
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()
	if strings.Contains(command, "git-upload-pack") {
		stream, err := repository.SSHServiceClient.SSHUploadPack(ctxWithCancel)
		if err != nil {
			_, _ = stderr.Read([]byte("Failed to create stream:" + err.Error()))
			return
		}

		req := &gitalypb.SSHUploadPackRequest{
			Repository: repository.GitalyRepo,
		}

		if err := stream.Send(req); err != nil {
			_, _ = stderr.Read([]byte("Failed to send request:" + err.Error()))
			return
		}

		numStreams := 1
		errC := make(chan error, numStreams)

		go func() {
			rr := streamio.NewReader(func() ([]byte, error) {
				response, err := stream.Recv()
				if err != nil {
					return nil, err
				}
				if response == nil {
					return nil, nil
				}
				return response.Stdout, err
			})
			_, err := io.Copy(session, rr)
			errC <- err
		}()

		go func() {
			sw := streamio.NewWriter(func(data []byte) error {
				return stream.Send(&gitalypb.SSHUploadPackRequest{Stdin: data})
			})
			_, err := io.Copy(sw, session)
			stream.CloseSend()
			errC <- err
		}()

		for i := 0; i < numStreams; i++ {
			if err := <-errC; err != nil {
				return
			}
		}
	} else if strings.Contains(command, "git-upload-archive") {
		stream, err := repository.SSHServiceClient.SSHUploadArchive(ctxWithCancel)
		if err != nil {
			_, _ = stderr.Read([]byte("Failed to create stream:" + err.Error()))
			return
		}

		req := &gitalypb.SSHUploadArchiveRequest{
			Repository: repository.GitalyRepo,
		}

		if err := stream.Send(req); err != nil {
			_, _ = stderr.Read([]byte("Failed to send request:" + err.Error()))
			return
		}

		numStreams := 1
		errC := make(chan error, numStreams)

		go func() {
			rr := streamio.NewReader(func() ([]byte, error) {
				response, err := stream.Recv()
				if err != nil {
					return nil, err
				}
				if response == nil {
					return nil, nil
				}
				return response.Stdout, err
			})
			_, err := io.Copy(session, rr)
			errC <- err
		}()

		go func() {
			sw := streamio.NewWriter(func(data []byte) error {
				return stream.Send(&gitalypb.SSHUploadArchiveRequest{Stdin: data})
			})
			_, err := io.Copy(sw, session)
			stream.CloseSend()
			errC <- err
		}()

		for i := 0; i < numStreams; i++ {
			if err := <-errC; err != nil {
				return
			}
		}
	} else {
		receivePackStream, err := repository.SSHServiceClient.SSHReceivePack(ctxWithCancel)
		if err != nil {
			_, _ = stderr.Read([]byte("Failed to create stream:" + err.Error()))
			return
		}

		req := &gitalypb.SSHReceivePackRequest{
			Repository:   repository.GitalyRepo,
			GlId:         keyOwnerID,
			GlRepository: reponame,
			GlUsername:   username,
		}
		if err := receivePackStream.Send(req); err != nil {
			_, _ = stderr.Read([]byte("Failed to send request:" + err.Error()))
			return
		}

		numStreams := 2
		errC := make(chan error, numStreams)

		go func() {
			rr := streamio.NewReader(func() ([]byte, error) {
				response, err := receivePackStream.Recv()
				if err != nil {
					return nil, err
				}
				if response == nil {
					return nil, nil
				}
				return response.Stdout, err
			})
			_, err := io.Copy(session, rr)
			errC <- err
		}()

		go func() {
			sw := streamio.NewWriter(func(data []byte) error {
				if string(data) == "0000" {
					return nil
				}
				return receivePackStream.Send(&gitalypb.SSHReceivePackRequest{Stdin: data})
			})
			_, err := io.Copy(sw, session)
			receivePackStream.CloseSend()
			errC <- err
		}()

		for i := 0; i < numStreams; i++ {
			if err := <-errC; err != nil {
				return
			}
		}

		needSettingDefaultBranch, err := repository.HasOnlyOneBranch()
		if err != nil {
			log.Error("Error has occurred while checking the number of branches on server: %v", err)
			return
		}

		if needSettingDefaultBranch {
			if err = utils.SetServerDefaultBranch(ctx, repository); err != nil {
				log.Error("Error has occurred while fixing server default branch: %v", err)
				return
			}
		}
	}

	if err := session.Exit(getExitStatusFromError(err)); err != nil && !errors.Is(err, io.EOF) {
		log.Error("Session failed to exit. %s", err)
	}
}

func publicKeyHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	if log.IsDebug() { // <- FingerprintSHA256 is kinda expensive so only calculate it if necessary
		log.Debug("Handle Public Key: Fingerprint: %s from %s", gossh.FingerprintSHA256(key), ctx.RemoteAddr())
	}

	if ctx.User() != setting.SSH.BuiltinServerUser {
		log.Warn("Invalid SSH username %s - must use %s for all git operations via ssh", ctx.User(), setting.SSH.BuiltinServerUser)
		log.Warn("Failed authentication attempt from %s", ctx.RemoteAddr())
		return false
	}

	// check if we have a certificate
	if cert, ok := key.(*gossh.Certificate); ok {
		if log.IsDebug() { // <- FingerprintSHA256 is kinda expensive so only calculate it if necessary
			log.Debug("Handle Certificate: %s Fingerprint: %s is a certificate", ctx.RemoteAddr(), gossh.FingerprintSHA256(key))
		}

		if len(setting.SSH.TrustedUserCAKeys) == 0 {
			log.Warn("Certificate Rejected: No trusted certificate authorities for this server")
			log.Warn("Failed authentication attempt from %s", ctx.RemoteAddr())
			return false
		}

		// look for the exact principal
	principalLoop:
		for _, principal := range cert.ValidPrincipals {
			pkey, err := asymkey_model.SearchPublicKeyByContentExact(ctx, principal)
			if err != nil {
				if asymkey_model.IsErrKeyNotExist(err) {
					log.Debug("Principal Rejected: %s Unknown Principal: %s", ctx.RemoteAddr(), principal)
					continue principalLoop
				}
				log.Error("SearchPublicKeyByContentExact: %v", err)
				return false
			}

			c := &gossh.CertChecker{
				IsUserAuthority: func(auth gossh.PublicKey) bool {
					marshaled := auth.Marshal()
					for _, k := range setting.SSH.TrustedUserCAKeysParsed {
						if bytes.Equal(marshaled, k.Marshal()) {
							return true
						}
					}

					return false
				},
			}

			// check the CA of the cert
			if !c.IsUserAuthority(cert.SignatureKey) {
				if log.IsDebug() {
					log.Debug("Principal Rejected: %s Untrusted Authority Signature Fingerprint %s for Principal: %s", ctx.RemoteAddr(), gossh.FingerprintSHA256(cert.SignatureKey), principal)
				}
				continue principalLoop
			}

			// validate the cert for this principal
			if err := c.CheckCert(principal, cert); err != nil {
				// User is presenting an invalid certificate - STOP any further processing
				log.Error("Invalid Certificate KeyID %s with Signature Fingerprint %s presented for Principal: %s from %s", cert.KeyId, gossh.FingerprintSHA256(cert.SignatureKey), principal, ctx.RemoteAddr())
				log.Warn("Failed authentication attempt from %s", ctx.RemoteAddr())

				return false
			}

			if log.IsDebug() { // <- FingerprintSHA256 is kinda expensive so only calculate it if necessary
				log.Debug("Successfully authenticated: %s Certificate Fingerprint: %s Principal: %s", ctx.RemoteAddr(), gossh.FingerprintSHA256(key), principal)
			}
			ctx.SetValue(scKeyID, pkey.ID)
			ctx.SetValue(scKeyOwnerID, pkey.OwnerID)

			return true
		}

		log.Warn("From %s Fingerprint: %s is a certificate, but no valid principals found", ctx.RemoteAddr(), gossh.FingerprintSHA256(key))
		log.Warn("Failed authentication attempt from %s", ctx.RemoteAddr())
		return false
	}

	if log.IsDebug() { // <- FingerprintSHA256 is kinda expensive so only calculate it if necessary
		log.Debug("Handle Public Key: %s Fingerprint: %s is not a certificate", ctx.RemoteAddr(), gossh.FingerprintSHA256(key))
	}

	pkey, err := asymkey_model.SearchPublicKeyByContent(ctx, strings.TrimSpace(string(gossh.MarshalAuthorizedKey(key))))
	if err != nil {
		if asymkey_model.IsErrKeyNotExist(err) {
			log.Warn("Unknown public key: %s from %s", gossh.FingerprintSHA256(key), ctx.RemoteAddr())
			log.Warn("Failed authentication attempt from %s", ctx.RemoteAddr())
			return false
		}
		log.Error("SearchPublicKeyByContent: %v", err)
		return false
	}

	if log.IsDebug() { // <- FingerprintSHA256 is kinda expensive so only calculate it if necessary
		log.Debug("Successfully authenticated: %s Public Key Fingerprint: %s", ctx.RemoteAddr(), gossh.FingerprintSHA256(key))
	}
	ctx.SetValue(scKeyID, pkey.ID)
	ctx.SetValue(scKeyOwnerID, pkey.OwnerID)

	return true
}

// sshConnectionFailed logs a failed connection
// -  this mainly exists to give a nice function name in logging
func sshConnectionFailed(conn net.Conn, err error) {
	// Log the underlying error with a specific message
	log.Warn("Failed connection from %s with error: %v", conn.RemoteAddr(), err)
	// Log with the standard failed authentication from message for simpler fail2ban configuration
	log.Warn("Failed authentication attempt from %s", conn.RemoteAddr())
}

// Listen starts a SSH server listens on given port.
func Listen(host string, port int, ciphers, keyExchanges, macs []string) {
	srv := ssh.Server{
		Addr:             net.JoinHostPort(host, strconv.Itoa(port)),
		PublicKeyHandler: publicKeyHandler,
		Handler:          sessionHandler,
		ServerConfigCallback: func(ctx ssh.Context) *gossh.ServerConfig {
			config := &gossh.ServerConfig{}
			config.KeyExchanges = keyExchanges
			config.MACs = macs
			config.Ciphers = ciphers
			return config
		},
		ConnectionFailedCallback: sshConnectionFailed,
		// We need to explicitly disable the PtyCallback so text displays
		// properly.
		PtyCallback: func(ctx ssh.Context, pty ssh.Pty) bool {
			return false
		},
	}

	keys := make([]string, 0, len(setting.SSH.ServerHostKeys))
	for _, key := range setting.SSH.ServerHostKeys {
		isExist, err := util.IsExist(key)
		if err != nil {
			log.Fatal("Unable to check if %s exists. Error: %v", setting.SSH.ServerHostKeys, err)
		}
		if isExist {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		filePath := filepath.Dir(setting.SSH.ServerHostKeys[0])

		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			log.Error("Failed to create dir %s: %v", filePath, err)
		}

		err := GenKeyPair(setting.SSH.ServerHostKeys[0])
		if err != nil {
			log.Fatal("Failed to generate private key: %v", err)
		}
		log.Trace("New private key is generated: %s", setting.SSH.ServerHostKeys[0])
		keys = append(keys, setting.SSH.ServerHostKeys[0])
	}

	for _, key := range keys {
		log.Info("Adding SSH host key: %s", key)
		err := srv.SetOption(ssh.HostKeyFile(key))
		if err != nil {
			log.Error("Failed to set Host Key. %s", err)
		}
	}

	go func() {
		_, _, finished := process.GetManager().AddTypedContext(graceful.GetManager().HammerContext(), "Service: Built-in SSH server", process.SystemProcessType, true)
		defer finished()
		listen(&srv)
	}()
}

// GenKeyPair make a pair of public and private keys for SSH access.
// Public key is encoded in the format for inclusion in an OpenSSH authorized_keys file.
// Private Key generated is PEM encoded
func GenKeyPair(keyPath string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	f, err := os.OpenFile(keyPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if err = f.Close(); err != nil {
			log.Error("Close: %v", err)
		}
	}()

	if err := pem.Encode(f, privateKeyPEM); err != nil {
		return err
	}

	// generate public key
	pub, err := gossh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	public := gossh.MarshalAuthorizedKey(pub)
	p, err := os.OpenFile(keyPath+".pub", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if err = p.Close(); err != nil {
			log.Error("Close: %v", err)
		}
	}()
	_, err = p.Write(public)
	return err
}
