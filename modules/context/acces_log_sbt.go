package context

import (
	"bytes"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web/middleware"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"time"
)

type routerLoggerOptionsSbt struct {
	req            *http.Request
	Identity       *string
	Start          *time.Time
	End            *time.Time
	Duration       *time.Duration
	Body           *string
	ResponseWriter http.ResponseWriter
	Ctx            map[string]interface{}
}

const traceIdCookieName = "traceId"

// AccessLoggerSbt returns a middleware to log access logger
// оригинальный обработчик тут modules/context/access_log.go
func AccessLoggerSbt() func(http.Handler) http.Handler {
	logger := log.GetLogger("access")
	logTemplate, _ := template.New("log").Parse(setting.Log.AccessLogTemplate)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()

			reqHost, _, err := net.SplitHostPort(req.RemoteAddr)
			if err != nil {
				reqHost = req.RemoteAddr
			}

			var bodyStr string
			userAgent := req.Header.Get("User-Agent")
			if strings.HasPrefix(strings.ToLower(userAgent), "git") {
				bodyStr = "*hidden git content*"
			} else {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					log.Error("Error reading request body, error: %v", err, userAgent)
				}
				req.Body = io.NopCloser(bytes.NewBuffer(body))

				if len(body) != 0 {
					bodyStr = string(body)

					var params map[string]interface{}
					//если тело в формате JSON
					err := json.Unmarshal(body, &params)

					if err != nil {
						//если тело в формате URL query
						params = make(map[string]interface{})
						gr := strings.Split(bodyStr, "&")
						for _, pair := range gr {
							pair := strings.Split(pair, "=")
							//если в теле непонятная структура, то выходим из цикла
							if len(gr) == 1 && len(pair) == 1 {
								break
							}
							if len(pair) > 1 {
								params[pair[0]] = pair[1]
							} else {
								params[pair[0]] = nil
							}
						}
					}

					if len(params) != 0 {
						omitSensitiveInfo(params)
						bodyBytes, _ := json.Marshal(params)
						bodyStr = string(bodyBytes)
					} else {
						//если в теле непонятная структура, то убираем переносы строк для экономии лога
						re := regexp.MustCompile(`\r?\n`)
						bodyStr = re.ReplaceAllString(bodyStr, " ")
					}
				} else {
					bodyStr = "-"
				}
				if len(bodyStr) > 1000 {
					bodyStr = bodyStr[1:999]
				}
			}

			traceId := uuid.New().String()
			cookie := http.Cookie{Name: traceIdCookieName, Value: traceId}
			req.AddCookie(&cookie)
			next.ServeHTTP(w, req)
			rw := w.(ResponseWriter)

			identity := "-"
			data := middleware.GetContextData(req.Context())
			if signedUser, ok := data[middleware.ContextDataKeySignedUser].(*userModel.User); ok {
				identity = signedUser.Name
			}

			buf := bytes.NewBuffer([]byte{})

			end := time.Now()
			duration := end.Sub(start)

			err = logTemplate.Execute(buf, routerLoggerOptionsSbt{
				req:            req,
				Identity:       &identity,
				Start:          &start,
				End:            &end,
				Duration:       &duration,
				Body:           &bodyStr,
				ResponseWriter: rw,
				Ctx: map[string]interface{}{
					"RemoteAddr": req.RemoteAddr,
					"RemoteHost": reqHost,
					"Req":        req,
				},
			})
			if err != nil {
				log.Error("Could not execute access logger template: %v", err.Error())
			}

			logger.Info("[TraceId: %s] %s", traceId, buf.String())
		})
	}
}

func omitSensitiveInfo(params map[string]interface{}) {
	hiddenFields := []string{"password", "retype", "image", "autofill_dummy_password", "old_password", "new_password"}

	for _, v := range hiddenFields {
		_, exists := params[v]
		if exists {
			params[v] = "***"
		}
	}
}
