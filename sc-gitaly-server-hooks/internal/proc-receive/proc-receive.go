package proc_receive

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"sc-gitaly-server-hooks/pkg/client"
	"sc-gitaly-server-hooks/pkg/logger"
	"sc-gitaly-server-hooks/pkg/models"
	"sc-gitaly-server-hooks/pkg/readers/console"
	"sc-gitaly-server-hooks/pkg/readers/env"
	"sc-gitaly-server-hooks/pkg/writers/pkt_line"
)

const VersionHead string = "version=1"

type ProcReceiveHook struct {
	hookLogger    *logger.HookLogger
	consoleReader console.Reader
	envReader     env.Reader
	pktLineWriter pkt_line.Writer
	client        client.HookClient
}

func NewProcReceiveHook(logger *logger.HookLogger, consoleReader console.Reader, envReader env.Reader, pktLineWriter pkt_line.Writer, client client.HookClient) ProcReceiveHook {
	return ProcReceiveHook{
		hookLogger:    logger,
		consoleReader: consoleReader,
		envReader:     envReader,
		pktLineWriter: pktLineWriter,
		client:        client,
	}
}

func (h ProcReceiveHook) Run(ctx context.Context) error {
	ownerName, err := h.envReader.GetByKey(models.EnvRepoUsername)
	if err != nil {
		h.hookLogger.Error("error getting owner name", err)
		return fmt.Errorf("run proc-receive hook: %w", err)
	}
	h.hookLogger.OwnerName = ownerName

	repoName, err := h.envReader.GetByKey(models.EnvRepoName)
	if err != nil {
		h.hookLogger.Error("error getting repo name", err)
		return fmt.Errorf("run proc-receive hook: %w", err)
	}
	h.hookLogger.RepoName = repoName

	pusherId, err := h.envReader.GetByKey(models.EnvPusherID)
	if err != nil {
		h.hookLogger.Error("error getting pusher id", err)
		return fmt.Errorf("run proc-receive hook: %w", err)
	}
	h.hookLogger.PusherId = pusherId

	pusherName, err := h.envReader.GetByKey(models.EnvPusherName)
	if err != nil {
		h.hookLogger.Error("error getting pusher name", err)
		return fmt.Errorf("run proc-receive hook: %w", err)
	}

	userID, err := strconv.ParseInt(pusherId, 10, 64)
	if err != nil {
		h.hookLogger.Error("error parsing pusher id", err)
		return fmt.Errorf("run proc-receive hook: %w", err)
	}

	// 1. Version and features negotiation.
	// S: PKT-LINE(version=1\0push-options atomic...) / PKT-LINE(version=1\n)
	// S: flush-pkt
	// H: PKT-LINE(version=1\0push-options...)
	// H: flush-pkt
	rs, err := h.consoleReader.ReadPktLine(ctx, os.Stdin, models.PktLineTypeData)
	if err != nil {
		h.hookLogger.Error("error reading pkt-line", err)
		return fmt.Errorf("run proc-receive hook: %w", err)
	}

	var (
		hasPushOptions bool
		response       = []byte(VersionHead)
		requestOptions []string
	)

	index := bytes.IndexByte(rs.Data, byte(0))
	if index >= len(rs.Data) {
		err = fmt.Errorf("Protocol: format error.\npkt-line: format error %s", rs.Data)
		h.hookLogger.Error("error indexing byte", err)
		return fmt.Errorf("run proc-receive hook: %w", err)
	}

	if index < 0 {
		if len(rs.Data) != 10 || rs.Data[9] != '\n' {
			err = fmt.Errorf("Protocol: format error.\npkt-line: format error %s", rs.Data)
			h.hookLogger.Error("incorrect format data", err)
			return fmt.Errorf("run proc-receive hook: %w", err)
		}
		index = 9
	}

	if string(rs.Data[0:index]) != VersionHead {
		err = fmt.Errorf("Protocol: version error.\nReceived unsupported version: %s", string(rs.Data[0:index]))
		h.hookLogger.Error("unsupported version", err)
		return fmt.Errorf("run proc-receive hook: %w", err)
	}

	requestOptions = strings.Split(string(rs.Data[index+1:]), " ")

	for _, option := range requestOptions {
		if strings.HasPrefix(option, "push-options") {
			response = append(response, byte(0))
			response = append(response, []byte("push-options")...)
			hasPushOptions = true
		}
	}
	response = append(response, '\n')

	if _, err = h.consoleReader.ReadPktLine(ctx, os.Stdin, models.PktLineTypeFlush); err != nil {
		h.hookLogger.Error("error reading pkt-line", err)
		return fmt.Errorf("run proc-receive hook: %w", err)
	}

	if err = h.pktLineWriter.WriteDataPktLine(os.Stdout, response); err != nil {
		h.hookLogger.Error("error writing data pkt-line", err)
		return fmt.Errorf("run proc-receive hook: %w", err)
	}

	if err = h.pktLineWriter.WriteFlushPktLine(os.Stdout); err != nil {
		h.hookLogger.Error("error writing flush pkt-line", err)
		return fmt.Errorf("run proc-receive hook: %w", err)
	}

	// 2. receive commands from server.
	// S: PKT-LINE(<old-oid> <new-oid> <ref>)
	// S: ... ...
	// S: flush-pkt
	// # [receive push-options]
	// S: PKT-LINE(push-option)
	// S: ... ...
	// S: flush-pkt
	hookOptions := models.NewHookOptionsWithUserInfo(userID, pusherName)

	for {
		// pktLineTypeUnknow means pktLineTypeFlush and pktLineTypeData all allowed
		rs, err = h.consoleReader.ReadPktLine(ctx, os.Stdout, models.PktLineTypeUnknow)
		if err != nil {
			h.hookLogger.Error("error reading pkt-line", err)
			return fmt.Errorf("run proc-receive hook: %w", err)
		}

		if rs.Type == models.PktLineTypeFlush {
			break
		}
		t := strings.SplitN(string(rs.Data), " ", 3)
		if len(t) != 3 {
			continue
		}
		hookOptions.OldCommitIDs = append(hookOptions.OldCommitIDs, t[0])
		hookOptions.NewCommitIDs = append(hookOptions.NewCommitIDs, t[1])
		hookOptions.RefFullNames = append(hookOptions.RefFullNames, t[2])
	}

	hookOptions.GitPushOptions = make(map[string]string)

	if hasPushOptions {
		for {
			rs, err = h.consoleReader.ReadPktLine(ctx, os.Stdout, models.PktLineTypeUnknow)
			if err != nil {
				h.hookLogger.Error("error reading pkt-line", err)
				return fmt.Errorf("run proc-receive hook: %w", err)
			}

			if rs.Type == models.PktLineTypeFlush {
				break
			}

			kv := strings.SplitN(string(rs.Data), "=", 2)
			if len(kv) == 2 {
				hookOptions.GitPushOptions[kv[0]] = kv[1]
			}
		}
	}

	requestOpts := models.NewHookRequestOptions(ownerName, repoName, hookOptions)

	// 3. run hook
	resp, extra := h.client.ProcReceive(ctx, requestOpts)
	if extra.HasError() {
		h.hookLogger.Error("error request hook on client", err)
		return fmt.Errorf("run proc-receive hook: %w", err)
	}

	// 4. response result to service
	// # a. OK, but has an alternate reference.  The alternate reference name
	// # and other status can be given in option directives.
	// H: PKT-LINE(ok <ref>)
	// H: PKT-LINE(option refname <refname>)
	// H: PKT-LINE(option old-oid <old-oid>)
	// H: PKT-LINE(option new-oid <new-oid>)
	// H: PKT-LINE(option forced-update)
	// H: ... ...
	// H: flush-pkt
	// # b. NO, I reject it.
	// H: PKT-LINE(ng <ref> <reason>)
	// # c. Fall through, let 'receive-pack' to execute it.
	// H: PKT-LINE(ok <ref>)
	// H: PKT-LINE(option fall-through)
	for _, rs := range resp.Results {
		if len(rs.Err) > 0 {
			if err = h.pktLineWriter.WriteDataPktLine(os.Stdout, []byte("ng "+rs.OriginalRef+" "+rs.Err)); err != nil {
				h.hookLogger.Error("error writing data pkt-line", err)
				return fmt.Errorf("run proc-receive hook: %w", err)
			}
			continue
		}

		if rs.IsNotMatched {
			if err = h.pktLineWriter.WriteDataPktLine(os.Stdout, []byte("ok "+rs.OriginalRef)); err != nil {
				h.hookLogger.Error("error writing data pkt-line", err)
				return fmt.Errorf("run proc-receive hook: %w", err)
			}

			if err = h.pktLineWriter.WriteDataPktLine(os.Stdout, []byte("option fall-through")); err != nil {
				h.hookLogger.Error("error writing data pkt-line", err)
				return fmt.Errorf("run proc-receive hook: %w", err)
			}
			continue
		}

		if err = h.pktLineWriter.WriteDataPktLine(os.Stdout, []byte("ok "+rs.OriginalRef)); err != nil {
			h.hookLogger.Error("error writing data pkt-line", err)
			return fmt.Errorf("run proc-receive hook: %w", err)
		}

		if err = h.pktLineWriter.WriteDataPktLine(os.Stdout, []byte("option refname "+rs.Ref)); err != nil {
			h.hookLogger.Error("error writing data pkt-line", err)
			return fmt.Errorf("run proc-receive hook: %w", err)
		}
		if rs.OldOID != models.EmptySHA {
			if err = h.pktLineWriter.WriteDataPktLine(os.Stdout, []byte("option old-oid "+rs.OldOID)); err != nil {
				h.hookLogger.Error("error writing data pkt-line", err)
				return fmt.Errorf("run proc-receive hook: %w", err)
			}
		}
		if err = h.pktLineWriter.WriteDataPktLine(os.Stdout, []byte("option new-oid "+rs.NewOID)); err != nil {
			h.hookLogger.Error("error writing data pkt-line", err)
			return fmt.Errorf("run proc-receive hook: %w", err)
		}
		if rs.IsForcePush {
			if err = h.pktLineWriter.WriteDataPktLine(os.Stdout, []byte("option forced-update")); err != nil {
				h.hookLogger.Error("error writing data pkt-line", err)
				return fmt.Errorf("run proc-receive hook: %w", err)
			}
		}
	}
	if err = h.pktLineWriter.WriteFlushPktLine(os.Stdout); err != nil {
		h.hookLogger.Error("error writing flush pkt-line", err)
		return fmt.Errorf("run proc-receive hook: %w", err)
	}

	h.hookLogger.Debug("Hook success finished")

	return nil
}
