package setting

import (
	"fmt"
	"regexp"
)

// ZERO_COUNT_REGEX регулярка для проверки количества 0 в начале строки
const ZERO_COUNT_REGEX = "^0{%d}\\w"

var (
	//EnableProofOfWork Флаг включения режима проверки Proof-Of-Work для запросов на регистрацию и аутентификацию
	EnableProofOfWork bool
	// ZeroCount Количество нулей в начале строки Proof-Of-Work
	ZeroCount           int
	RegexpWithZeroCount *regexp.Regexp
)

// https://dzo.sw.sbc.space/wiki/display/GITRU/Proof-Of-Work
func loadProofOfWorkFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("proofOfWork")
	EnableProofOfWork = sec.Key("ENABLE_PROOF_OF_WORK").MustBool(false)
	ZeroCount = sec.Key("ZERO_COUNT").MustInt(1)
	RegexpWithZeroCount = regexp.MustCompile(fmt.Sprintf(ZERO_COUNT_REGEX, ZeroCount))
}
