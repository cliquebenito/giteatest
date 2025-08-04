package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/util"
)

// GenerateTlsConfigForMTLS создаем новый tls config с mtls certificates для client
func GenerateTlsConfigForMTLS(nameMTLSConn string, mtlsCert, mtlsKey []byte, caCerts [][]byte) *tls.Config {
	auditKey := util.CheckMTLSParameterForAudit(nameMTLSConn)
	auditParams := map[string]string{
		auditKey: nameMTLSConn,
	}

	if len(caCerts) == 0 || len(mtlsCert) == 0 || len(mtlsKey) == 0 {
		auditParams["error"] = "Error has occurred while getting CA cert, MTLS cert or MTLS key is empty"
		audit.CreateAndSendEvent(audit.TLSConfigCreatingWithMTLS, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Fatal("Error has occurred while getting CA cert, MTLS cert or MTLS key is empty")
	}
	cert, err := tls.X509KeyPair(mtlsCert, mtlsKey)
	if err != nil {
		auditParams["error"] = fmt.Sprintf("Error has occurred while trying to create TLS by a cert and a key for %s: %v", nameMTLSConn, err)
		audit.CreateAndSendEvent(audit.TLSConfigCreatingWithMTLS, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Fatal("Error has occurred while trying to create TLS by a cert and a key: %v", err)
	}
	// Создаем пул доверенных сертификатов
	caPool := x509.NewCertPool()
	for _, caCert := range caCerts {
		caPool.AppendCertsFromPEM(caCert)
	}
	// Настраиваем TLS конфигурацию
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool, // Указываем пул доверенных CA
	}
	audit.CreateAndSendEvent(audit.TLSConfigCreatingWithMTLS, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
	return config
}
