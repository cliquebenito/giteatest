package casbinlogger

import (
	"fmt"
	"strings"

	"code.gitea.io/gitea/modules/log"
)

// Logger allows to log casbin events.
type Logger struct {
	enabled bool
	logger  log.Logger
}

// New creates casbin logger.
func New() *Logger {
	logger := log.GetLogger("casbin")
	return &Logger{enabled: true, logger: logger}
}

// EnableLog allows to turn logging on and off.
func (l *Logger) EnableLog(enable bool) {
	l.enabled = enable
}

// IsEnabled allows to check that logger is enabled.
func (l *Logger) IsEnabled() bool {
	return l.enabled
}

// LogModel logs the model information.
func (l *Logger) LogModel(model [][]string) {
	if !l.enabled {
		return
	}

	var str strings.Builder
	str.WriteString("Model: ")

	for _, v := range model {
		str.WriteString(fmt.Sprintf("%v\n", v))
	}

	l.logger.Debug(str.String())
}

// LogEnforce logs the enforcer information.
func (l *Logger) LogEnforce(matcher string, request []interface{}, result bool, explains [][]string) {
	if !l.enabled {
		return
	}

	var reqStr strings.Builder
	reqStr.WriteString("Request: ")

	for i, rval := range request {
		if i != len(request)-1 {
			reqStr.WriteString(fmt.Sprintf("%v, ", rval))
		} else {
			reqStr.WriteString(fmt.Sprintf("%v", rval))
		}
	}
	reqStr.WriteString(fmt.Sprintf(" ---> %t\n", result))

	l.logger.Debug(reqStr.String())
}

// LogPolicy logs the policy information.
func (l *Logger) LogPolicy(policy map[string][][]string) {
	if !l.enabled {
		return
	}

	var str strings.Builder

	str.WriteString("Policy: ")

	for k, v := range policy {
		if k == "g" || k == "g2" {
			continue
		}
		str.WriteString(fmt.Sprintf("%s : %v\n", k, v))
	}

	l.logger.Debug(str.String())
}

// LogRole log info related to role.
func (l *Logger) LogRole(roles []string) {
	if !l.enabled {
		return
	}

	l.logger.Debug("Roles: ", strings.Join(roles, "\n"))
}

// LogError logs the error information.
func (l *Logger) LogError(err error, msg ...string) {
	if !l.enabled {
		return
	}

	l.logger.Error("message: %s, error: %s", msg, err)
}
