package mtls

import (
	"fmt"
	"strings"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
)

// TODO выпилить со влитием VCS-1684
const secretName = "secret_name"

// mLTSClientCertificate структура с client certificates
type mLTSClientCertificate struct {
	Cert, CertKey []byte
	CaCerts       [][]byte
}

// GetMTLSCertsFromSecMan получаем mtls certs для из Sec Man
func GetMTLSCertsFromSecMan(nameMTLSConn string, secManClient setting.GetCredSecMan) *mLTSClientCertificate {
	auditKey := util.CheckMTLSParameterForAudit(nameMTLSConn)
	auditParams := map[string]string{
		auditKey: nameMTLSConn,
	}

	mtlsConfig := setting.MTLSConnectionAvailable[nameMTLSConn]
	if mtlsConfig == nil || !mtlsConfig.Enabled {
		auditParams["error"] = fmt.Sprintf("Error has occured while establishing mtls connection for %s is not available", nameMTLSConn)
		audit.CreateAndSendEvent(audit.MTLSCredsGetFromSecMan, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Info("Params for sec man store weren't put in app.ini config for client: %s", nameMTLSConn)
		return nil
	}

	configForKvGet := &vault_model.KeyValueConfigForGetSecrets{
		SecretPath:  strings.TrimSpace(mtlsConfig.SecretPath),
		StoragePath: strings.TrimSpace(mtlsConfig.StoragePath),
		VersionKey:  mtlsConfig.VersionKey,
	}

	resp, err := secManClient.GetCredFromSecManByVersionKey(configForKvGet)
	if err != nil {
		auditParams["error"] = fmt.Sprintf("Error has occurred while trying to get cred from sec man for %s: %v", nameMTLSConn, err)
		audit.CreateAndSendEvent(audit.MTLSCredsGetFromSecMan, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Fatal("Error has occurred while trying to get cred from sec man: %v", err)
	}

	mtlsClientCert := &mLTSClientCertificate{}
	caCertNames := strings.Split(mtlsConfig.CaCertPath, ",")
	mtlsClientCert.CaCerts = make([][]byte, 0, len(caCertNames))
	for _, caCertName := range caCertNames {
		caCert := strings.TrimSpace(caCertName)
		if resp.Data[caCert] != "" {
			mtlsClientCert.CaCerts = append(mtlsClientCert.CaCerts, []byte(resp.Data[caCert]))
		} else {
			auditParams["error"] = fmt.Sprintf("Error has occurred while trying to get ca.cert for %s: %v", nameMTLSConn, err)
			audit.CreateAndSendEvent(audit.MTLSCredsGetFromSecMan, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
			log.Fatal("Error has occurred while trying to get ca.cert for %s: %v", nameMTLSConn, err)
		}
		auditParams[secretName] = caCert
		audit.CreateAndSendEvent(audit.MTLSCredsGetFromSecMan, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
	}
	if len(mtlsClientCert.CaCerts) == 0 {
		auditParams["error"] = fmt.Sprintf("Error has occurred because array with ca certs is empty for config %s: %v", nameMTLSConn, err)
		audit.CreateAndSendEvent(audit.MTLSCredsGetFromSecMan, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Fatal("Error has occurred while trying to get ca.cert for %s: %v", nameMTLSConn, err)
	}

	if setting.GetResponseNotNil(resp) && resp.Data[mtlsConfig.CertPath] != "" {
		mtlsClientCert.Cert = []byte(resp.Data[mtlsConfig.CertPath])
		auditParams[secretName] = mtlsConfig.CertPath
		audit.CreateAndSendEvent(audit.MTLSCredsGetFromSecMan, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
	} else {
		auditParams["error"] = fmt.Sprintf("Error has occurred while trying to get cert.crt for %s: %v", nameMTLSConn, err)
		audit.CreateAndSendEvent(audit.MTLSCredsGetFromSecMan, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Fatal("Error has occurred while trying to get cert for %s: %v", nameMTLSConn, err)
	}

	if resp.Data[mtlsConfig.KeyPath] != "" {
		mtlsClientCert.CertKey = []byte(resp.Data[mtlsConfig.KeyPath])
		auditParams[secretName] = mtlsConfig.KeyPath
		audit.CreateAndSendEvent(audit.MTLSCredsGetFromSecMan, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
	} else {
		auditParams["error"] = fmt.Sprintf("Error has occurred while trying to get cert.key for %s: %v", nameMTLSConn, err)
		audit.CreateAndSendEvent(audit.MTLSCredsGetFromSecMan, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Fatal("Error has occurred while trying to get cert.key for %s: %v", nameMTLSConn, err)
	}
	return mtlsClientCert
}
