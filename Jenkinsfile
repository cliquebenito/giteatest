def onDistrib(app, distr) {
// Set build Tools
    def goTool = tool name: "go-1.22", type: 'go'
    def nodeJsTool = tool name: 'v16.20.0-linux-x64', type: 'nodejs'
    def scannerTool = tool name: 'SonarQube_Scanner', type: 'hudson.plugins.sonar.SonarRunnerInstallation'

// Set build ENVs
    env.EXECUTABLE = "sc"
    env.EXTRA_GOFLAGS = "-trimpath"
    env.GITEA_VERSION = "${app.version}-${BUILD_NUMBER} (${app.branch})"
    env.GOSUMDB = "off"
    env.GO111MODULE = "on"
    env.TAGS = "netgo osusergo bindata"
    env.TEST_TAGS = "netgo osusergo bindata"
    env.GOPATH = "${goTool}/bin"
    env.PATH = "${env.PATH}:${goTool}/bin:${scannerTool}:${nodeJsTool}/bin/node"
    withCredentials([usernamePassword(credentialsId: "tuz_sbt_ci_gitru_osc", usernameVariable: "OSC_USER", passwordVariable: "OSC_TOKEN")]) {
        env.GOPROXY = "https://${OSC_USER}:${OSC_TOKEN}@sberworks.ru/osc/repo/go"
    }

// Cleanup before build
    withEnv(["GOROOT=${goTool}", "PATH+GO=${goTool}/bin"]) {
        sh "make clean-all"
    }

// Run unit-tests (backend)
    if (params.ADDITIONAL.contains('run-unit-tests')) {
        withEnv(["CGO_ENABLED=1", "GOROOT=${goTool}", "PATH+GO=${goTool}/bin"]) {
            catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE') {
                sh "go run gotest.tools/gotestsum@latest \
                        --junitfile-testcase-classname=short \
                        --junitfile-testsuite-name=relative \
                        --format pkgname \
                        --junitfile report-backend.xml \
                        -- -failfast -race -coverprofile=coverage.out ./..."
            }
        }
        if (fileExists('report-backend.xml')) {
            junit 'report-backend.xml'
       }
    }

// Run SonarQube scan
    if (params.ADDITIONAL.contains('run-sonar')) {
        withSonarQubeEnv(credentialsId: "tuz_sbt_ci_gitru_sonar", installationName: 'SonarQube') {
            sh """
                sonar-scanner \
                -Dsonar.projectKey=${app.sonar_project} \
                -Dsonar.projectVersion=${app.version} \
                -Dsonar.branch.name=${app.branch} \
                -Dsonar.nodejs.executable=${nodeJsTool}/bin/node \
                -Dsonar.go.coverage.reportPaths=coverage.out \
                -Dsonar.test.inclusions=**/*_test.go \
                -Dsonar.exclusions=tests/**,docs/**,contrib/**,docker/**,custom/**,public/**,**/vendor/**,**/*_test.go,*.*
            """
        }
    }

// Build Frontend
    nodejs(configId: 'tuz_npm_config_osc', nodeJSInstallationName: 'v16.20.0-linux-x64') {
        sh "make frontend"
    }

// Build Backend
    withEnv(["CGO_ENABLED=1", "GOROOT=${goTool}", "PATH+GO=${goTool}/bin"]) {
        sh "make backend"
    }
// Build environment-to-ini tool
    withEnv(["CGO_ENABLED=1", "GOROOT=${goTool}", "PATH+GO=${goTool}/bin"]) {
        sh "go build contrib/environment-to-ini/environment-to-ini.go"
    }

// Prepare docker artefacts
    sh "mv -f Dockerfile.sc Dockerfile"

// Add binary artifacts to distro structure
    distr.addBH("sc")
    distr.addBH("sc-gitaly-backup")
    distr.addConf("custom/conf/app.example.ini")

// Add docker artefacts to distro structure
    if (params.PLAYBOOKS.contains('docker')) {
        distr.addDockerItems ("gitt", "Dockerfile")
        distr.addDockerItems ("gitt", "docker/rootless")
        distr.addDockerItems ("gitt", "environment-to-ini")
        distr.addDockerItems ("gitt", "contrib/autocompletion/bash_autocomplete")
    }
}

return wrapJenkinsfile(this)
