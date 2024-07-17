void updateGitHubStatus(String context, String state, String description, String commitSHA) {
    withCredentials([usernamePassword(credentialsId: GITHUB_CREDENTIALS_ID, usernameVariable: 'GITHUB_USERNAME', passwordVariable: 'GITHUB_TOKEN')]) {
        echo "Updating GitHub status: context=${context}, state=${state}, description=${description}, commit=${commitSHA}"
        def payload = """
            {
                "state": "${state}",
                "target_url": "${env.BUILD_URL}",
                "description": "${description}",
                "context": "${context}"
            }
        """
        echo "Payload: ${payload}"
        def response = sh(script: "curl -H 'Authorization: token ${GITHUB_TOKEN}' -H 'Content-Type: application/json' -X POST -d '${payload}' ${GITHUB_API_URL}/${GITHUB_REPO_OWNER}/${GITHUB_REPO_NAME}/statuses/${commitSHA}", returnStdout: true).trim()
        echo "GitHub API response: ${response}"
        
        if (response.contains("error")) {
            error("GitHub status update failed: ${response}")
        } else {
            echo "GitHub status update successful: ${response}"
        }
    }
}

pipeline {
    agent any

    environment {
        GITHUB_CREDENTIALS_ID = 'github'
        GITHUB_REPO_OWNER = 'cyse7125-su24-team15'
        GITHUB_REPO_NAME = 'webapp-cve-consumer'
        GITHUB_API_URL = 'https://api.github.com/repos'
    }

    stages {
        stage('Initialize') {
            steps {
                script {
                    withCredentials([usernamePassword(credentialsId: GITHUB_CREDENTIALS_ID, usernameVariable: 'GITHUB_USERNAME', passwordVariable: 'GITHUB_TOKEN')]) {
                        def prCommitSHA = sh(script: "git ls-remote https://${GITHUB_USERNAME}:${GITHUB_TOKEN}@github.com/${GITHUB_REPO_OWNER}/${GITHUB_REPO_NAME}.git refs/pull/${env.CHANGE_ID}/head | cut -f1", returnStdout: true).trim()
                        echo "PR Commit SHA: ${prCommitSHA}"
                        env.PR_COMMIT_SHA = prCommitSHA
                    }
                }
            }
        }

        stage('Checkout') {
            steps {
                script {
                    echo 'Checking out the repository'
                    checkout([
                        $class: 'GitSCM',
                        branches: [[name: "FETCH_HEAD"]],
                        doGenerateSubmoduleConfigurations: false,
                        extensions: [],
                        userRemoteConfigs: [[
                            url: "https://github.com/${env.GITHUB_REPO_OWNER}/${env.GITHUB_REPO_NAME}.git",
                            refspec: "+refs/pull/${env.CHANGE_ID}/head:refs/remotes/origin/PR-${env.CHANGE_ID}",
                            credentialsId: env.GITHUB_CREDENTIALS_ID
                        ]]
                    ])
                }
            }
        }

        stage('Fetch Base Branch') {
            steps {
                script {
                    echo 'Fetching base branch from original repository'
                    withCredentials([usernamePassword(credentialsId: GITHUB_CREDENTIALS_ID, usernameVariable: 'GITHUB_USERNAME', passwordVariable: 'GITHUB_TOKEN')]) {
                        sh '''
                            git remote add upstream https://$GITHUB_USERNAME:$GITHUB_TOKEN@github.com/$GITHUB_REPO_OWNER/$GITHUB_REPO_NAME.git || true
                            git fetch upstream main
                        '''
                    }
                }
            }
        }

        stage('Create Commitlint Config') {
            steps {
                script {
                    echo 'Creating commitlint config...'
                    sh '''
                        mkdir -p /tmp/commitlint-config
                        echo "module.exports = { extends: ['$(npm root -g)/@commitlint/config-conventional/lib/index.js'] };" > /tmp/commitlint-config/commitlint.config.js
                    '''
                }
            }
        }

        stage('Check Conventional Commits') {
            steps {
                script {
                    echo 'Checking Conventional Commits'
                    withCredentials([usernamePassword(credentialsId: GITHUB_CREDENTIALS_ID, usernameVariable: 'GITHUB_USERNAME', passwordVariable: 'GITHUB_TOKEN')]) {
                        sh """
                            git fetch https://${GITHUB_USERNAME}:${GITHUB_TOKEN}@github.com/${GITHUB_REPO_OWNER}/${GITHUB_REPO_NAME}.git +refs/pull/${env.CHANGE_ID}/head:refs/remotes/origin/PR-${env.CHANGE_ID}
                            git fetch https://${GITHUB_USERNAME}:${GITHUB_TOKEN}@github.com/${GITHUB_REPO_OWNER}/${GITHUB_REPO_NAME}.git +refs/heads/${env.CHANGE_TARGET}:refs/remotes/origin/${env.CHANGE_TARGET}
                        """

                        def commits = sh(script: "git log --pretty=format:'%s' origin/${env.CHANGE_TARGET}..origin/PR-${env.CHANGE_ID}", returnStdout: true).trim().split('\n')
                        if (commits.size() == 1 && commits[0].isEmpty()) {
                            echo 'No new commits to check'
                        } else {
                            echo "Commits to be checked: ${commits}"
                            def hasErrors = false
                            commits.each { commit ->
                                def result = sh(script: "echo '${commit}' | commitlint --config /tmp/commitlint-config/commitlint.config.js", returnStatus: true)
                                if (result != 0) {
                                    echo "Commit message failed: ${commit}"
                                    hasErrors = true
                                }
                            }
                            if (hasErrors) {
                                updateGitHubStatus('conventional-commits', 'failure', 'Conventional Commits check failed', env.PR_COMMIT_SHA)
                                error('Conventional Commits check failed!')
                            }
                        }
                        updateGitHubStatus('conventional-commits', 'success', 'Conventional Commits check passed', env.PR_COMMIT_SHA)
                    }
                }
            }
        }
    }

    post {
        always {
            script {
                echo 'Cleaning up'
                deleteDir()
            }
        }
    }
}