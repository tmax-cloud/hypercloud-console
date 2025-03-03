pipeline {
  parameters {
    choice(name: 'BUILD_MODE', choices:['PATCH','HOTFIX','IMAGE'], description: 'Select the mode you want to act')
    choice(name: 'DEPLOY', choices:['ck2-1', 'ck1-1', 'keycloak'], description: 'Select k8s env you want to deploy the console')

    string(name: 'KEYCLOAK', defaultValue: 'hyperauth.org', description: 'hyperauth url for login')
    string(name: 'REALM', defaultValue: 'tmax', description: 'hyperauth realm info')
    string(name: 'CLIENTID', defaultValue: 'ck-integration-hypercloud5', description: 'hyperauth client id info')
    string(name: 'MC_MODE', defaultValue: 'true', description: 'Choice multi cluster mode')    
  }
  triggers {
    // ref https://plugins.jenkins.io/parameterized-scheduler/
    // trigger at 9:00 every Thursday 
    parameterizedCron('''
    0 9 * * 4 %BUILD_MODE=PATCH
    ''')
    
  }
  environment { 
    BRANCH = "hc-dev-v5.0"
    BUILD_MODE = "${params.BUILD_MODE}"
    DEPLOY = "${params.DEPLOY}"

    DOCKER_REGISTRY="tmaxcloudck"
    PRODUCT = "hypercloud-console"
    MAJOR_VER="5"
    MINOR_VER="1"
    PATCH_VER="0"
    HOTFIX_VER="0"
    VER = "${MAJOR_VER}.${MINOR_VER}.${PATCH_VER}.${HOTFIX_VER}"

    // OPERATOR_VER = "5.1.0.1" // fixed at 5.1.0.1
    KEYCLOAK = "${params.KEYCLOAK}"
    REALM = "${params.REALM}"
    CLIENTID = "${params.CLIENTID}"
    MC_MODE = "${params.MC_MODE}"

    GUIDE_URL = "https://github.com/tmax-cloud/install-console/blob/5.0/README.md"

  }
  agent {
    kubernetes {
      cloud 'ck1-1'
      // yamlFile './KubernetesPod.yaml'
      yaml '''\
        apiVersion: v1
        kind: Pod      
        metadata:
          labels:
            some-label: some-label-value
            class: KubernetesDeclarativeAgentTest
        spec:
          containers:
          - name: docker 
            image: docker 
            command: 
            - cat 
            tty: true
            volumeMounts: 
            - mountPath: /var/run/docker.sock
              name: docker-volume
          - name: kubectl
            image: lachlanevenson/k8s-kubectl:v1.19.1
            command:
            - sh
            tty: true
          volumes:
          - name: docker-volume 
            hostPath: 
              path: /var/run/docker.sock 
              type: ""  
        '''.stripIndent()      
    }
  }

  stages {

    // When using SCM, the checkout stage can be completely omitted 
    stage('Git') {
      steps {
        git branch: "${BRANCH}", credentialsId: 'jinsoo-youn', url: 'https://github.com/tmax-cloud/hypercloud-console5.0.git'
        sh """
        git branch
        git pull origin HEAD:${BRANCH}
        """
         script {
            PATCH_VER = sh(script: 'cat ./CHANGELOG/tag.txt | head -2 | tail -1 | cut --delimiter="." --fields=3', returnStdout: true).trim()
            HOTFIX_VER = sh(script: 'cat ./CHANGELOG/tag.txt | head -2 | tail -1 | cut --delimiter="." --fields=4', returnStdout: true).trim()
          if (BUILD_MODE == 'PATCH') {
            PATCH_VER++
            HOTFIX_VER = "0"
            VER = "${MAJOR_VER}.${MINOR_VER}.${PATCH_VER}.${HOTFIX_VER}"
          } else if (BUILD_MODE == 'HOTFIX') {
            HOTFIX_VER++
            VER = "${MAJOR_VER}.${MINOR_VER}.${PATCH_VER}.${HOTFIX_VER}"
          }
        }
        withCredentials([usernamePassword(credentialsId: 'jinsoo-youn', usernameVariable: 'username', passwordVariable: 'password')]) {      
          sh """
            git config --global user.name ${username}
            git config --global user.email jinsoo_youn@tmax.co.kr
            git config --global credential.username ${username}
            git config --global credential.helper "!echo password=${password}; echo"          
          """        
          sh """
          git tag ${VER}
          git push origin HEAD:${BRANCH} --tags
          echo "Console Version History" > ./CHANGELOG/tag.txt
          git tag --list "5.1.*" --sort=-version:refname >> ./CHANGELOG/tag.txt
          git add -A
          git commit -m '[FEAT] ADD the console version ${PRODUCT}_${VER} on tag.txt'
          git push origin HEAD:${BRANCH}
          """
        }
      }
    }

    stage('Build') {
      steps{
        container('docker'){
          withCredentials([usernamePassword(
            credentialsId: 'tmaxcloudck',
            usernameVariable: 'DOCKER_USER',
            passwordVariable: 'DOCKER_PWD')]){
            sh "docker login -u ${DOCKER_USER} -p ${DOCKER_PWD}"
            sh "docker build -t ${DOCKER_REGISTRY}/${PRODUCT}:${VER} -f ./Dockerfile ."
            sh "docker push ${DOCKER_REGISTRY}/${PRODUCT}:${VER}"
          }          
        }
      }
    }

    stage('Deploy') {
      when {
        anyOf {
          environment name: 'BUILD_MODE', value: 'PATCH'
          environment name: 'BUILD_MODE', value: 'HOTFIX'
        }
      }
      steps {
        container('kubectl') {
          withKubeConfig([credentialsId: "${DEPLOY}"]) {
          sh "./install.sh"
          }
        }
      }
    }
        
    stage('Changelog'){
      when {
        anyOf {
          environment name: 'BUILD_MODE', value: 'PATCH'
          environment name: 'BUILD_MODE', value: 'HOTFIX'
        }
      }
      steps {
        script {
          if (BUILD_MODE == 'PATCH') {
            // TEMP = (("${params.PATCH_VER}" as int) -1).toString()
            PATCH_VER--
            HOTFIX_VER = "0"
            PRE_VER = "${MAJOR_VER}.${MINOR_VER}.${PATCH_VER}.${HOTFIX_VER}"
          } else if (BUILD_MODE == 'HOTFIX') {
            // TEMP = (("${params.HOTFIX_VER}" as int) -1).toString()
            HOTFIX_VER--
            PRE_VER = "${MAJOR_VER}.${MINOR_VER}.${PATCH_VER}.${HOTFIX_VER}"
          }
        }
        withCredentials([usernamePassword(credentialsId: 'jinsoo-youn', usernameVariable: 'username', passwordVariable: 'password')]) {      
          sh """
            git config --global user.name ${username}
            git config --global user.email jinsoo_youn@tmax.co.kr
            git config --global credential.username ${username}
            git config --global credential.helper "!echo password=${password}; echo"          
          """ 
          // Creat CHANGELOG-${VER}.md
          sh """
            echo '# hypercloud-console patch note' > ./CHANGELOG/CHANGELOG-${VER}.md
            echo '## hypercloud-console_[major].[minor].[patch].[hotfix]' >> ./CHANGELOG/CHANGELOG-${VER}.md
            echo 'Version: ${PRODUCT}_${VER}' >> ./CHANGELOG/CHANGELOG-${VER}.md
            date '+%F  %r' >> ./CHANGELOG/CHANGELOG-${VER}.md
            git log --grep=[patch] -F --all-match --no-merges --date-order --reverse \
            --pretty=format:\"- %s (%cn) %n    Message: %b\" \
            --simplify-merges ${PRE_VER}..${VER} \
            >> ./CHANGELOG/CHANGELOG-${VER}.md
          """
          sh "git add -A"
          sh "git commit -m '[FEAT] BUILD ${PRODUCT}_${VER}' "
          sh "git push origin HEAD:${BRANCH}"        
        }
      }
    }

    stage('Email'){
      when {
        anyOf {
          environment name: 'BUILD_MODE', value: 'PATCH'
          environment name: 'BUILD_MODE', value: 'HOTFIX'
        }
      }
      steps {
        emailext (
          to: 'cqa1@tmax.co.kr, ck1@tmax.co.kr, ck2@tmax.co.kr',
          subject: "[${PRODUCT}] Release Update - ${PRODUCT}:${VER}", 
          attachmentsPattern: "**/CHANGELOG/CHANGELOG-${VER}.md",
          body: "안녕하세요. \n\n${PRODUCT} Release Update 입니다. \n\n변경사항 파일로 첨부합니다. \n\n감사합니다.\n\n" +
                "※ 이미지 : ${DOCKER_REGISTRY}/${PRODUCT}:${VER} \n\n※ 설치 가이드 : ${GUIDE_URL} ",
          mimeType: 'text/plain'  
        )
      }
    }

  }

  post {
    success {
      sh "echo SUCCESSFUL"
      emailext (
        to: "jinsoo_youn@tmax.co.kr",
        subject: "SUCCESSFUL: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]'",
        body:  """<p>SUCCESSFUL: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]':</p>
            <p>Check console output at &QUOT;<a href='${env.BUILD_URL}'>${env.JOB_NAME} [${env.BUILD_NUMBER}]</a>&QUOT;</p>""",
        mimeType: 'text/html',    
      )
    } 
    failure {
      sh "echo FAILED"
      emailext (
        to: "jinsoo_youn@tmax.co.kr",
        subject: "FAILED: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]'",
        body: """<p>FAILED: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]':</p>
          <p>Check console output at &QUOT;<a href='${env.BUILD_URL}'>${env.JOB_NAME} [${env.BUILD_NUMBER}]</a>&QUOT;</p>""",
        mimeType: 'text/html'
      )
    }
  }
}