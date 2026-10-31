// sub2api 生产镜像构建流水线（Jenkins / linux-sub2api 香港构建节点）。
//
// 只负责「打包 + 投递」：构建 ghcr.io/go1c/sub2api:sha-<SHA>（main-kin 额外打 :kin），
// 经受限 SSH（kin 侧 forced command /opt/lumio/bin/jenkins-image-load.sh）直接
// docker load 进 lumio-kin。不推 GHCR，不经 CN 机器。
// 部署动作（预部署 dump、翻 SUB2API_IMAGE、up -d --pull never、健康检查、deploy-state
// 记账）仍由运维按 LumioServerWorkFlow 的 runbook 执行。
pipeline {
  agent { label 'sub2api' }

  options {
    timeout(time: 120, unit: 'MINUTES')
    disableConcurrentBuilds(abortPrevious: false)
    buildDiscarder(logRotator(daysToKeepStr: '14', numToKeepStr: '30'))
    skipDefaultCheckout()
  }

  parameters {
    string(name: 'COMMIT', defaultValue: 'dev',
           description: 'Go1c/sub2api 的分支名、tag 或完整 40 位 SHA；发布请固定完整 SHA')
    choice(name: 'STACK', choices: ['publish', 'main-kin'],
           description: 'publish：现网 api.lumio.games；main-kin：sidecar，额外打 :kin tag')
  }

  environment {
    REPO       = 'https://github.com/Go1c/sub2api.git'
    IMAGE_ROOT = 'ghcr.io/go1c/sub2api'
    KIN_HOST   = '156.238.225.247'
    KIN_PORT   = '11851'
    KIN_KEY    = "${HOME}/.ssh/kin_image_load_ed25519"
  }

  stages {
    stage('解析提交') {
      steps {
        sh label: 'resolve', script: '''#!/usr/bin/env bash
          set -euo pipefail
          ref="${COMMIT}"
          if [[ "$ref" =~ ^[0-9a-f]{40}$ ]]; then
            SHA="$ref"
          else
            SHA=$(git ls-remote "${REPO}" | awk -v r="$ref" '
              $2 == ("refs/heads/" r) && !h { h = $1 }
              $2 == ("refs/tags/" r)  && !t { t = $1 }
              END { print (h ? h : t) }')
          fi
          test -n "$SHA"
          echo "$SHA" > "${WORKSPACE}/sha.txt"
          echo "resolved ${ref} -> ${SHA}"
        '''
        script { env.SHA = readFile('sha.txt').trim() }
      }
    }

    stage('取源码') {
      steps {
        sh label: 'fetch', script: '''#!/usr/bin/env bash
          set -euo pipefail
          rm -rf "${WORKSPACE}/src" && mkdir "${WORKSPACE}/src"
          git init -q "${WORKSPACE}/src"
          git -C "${WORKSPACE}/src" remote add origin "${REPO}"
          git -C "${WORKSPACE}/src" fetch -q --depth=1 origin "${SHA}"
          git -C "${WORKSPACE}/src" checkout -q --detach FETCH_HEAD
          test "$(git -C "${WORKSPACE}/src" rev-parse HEAD)" = "${SHA}"
        '''
      }
    }

    stage('构建镜像') {
      steps {
        sh label: 'buildx', script: '''#!/usr/bin/env bash
          set -euo pipefail
          docker buildx create --name sub2api --driver docker-container >/dev/null 2>&1 || docker buildx use sub2api
          docker buildx use sub2api
          DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
          tags=( -t "${IMAGE_ROOT}:sha-${SHA}" )
          if [[ "${STACK}" == "main-kin" ]]; then
            tags+=( -t "${IMAGE_ROOT}:kin" )
          fi
          cd "${WORKSPACE}/src"
          # GOLANG_IMAGE 需 >= backend/go.mod 的 go 指令（main-kin 线要求 >= 1.27.3）。
          # NODE_OPTIONS / CI_INLINE_TYPECHECK 把前端压进预算（见 vite.config.ts）；
          # GOFLAGS=-p=1 串行编译 worker：GOMEMLIMIT 按进程生效，并行 worker 会让
          # 峰值 = N x GOMEMLIMIT（ent 包单 worker ~2.2G），串行后峰值 = 单包 + 基础设施。
          docker buildx build \
            --platform linux/amd64 \
            --provenance=false \
            "${tags[@]}" \
            --build-arg VERSION="${STACK}" \
            --build-arg COMMIT="${SHA}" \
            --build-arg DATE="${DATE}" \
            --build-arg GOPROXY=https://proxy.golang.org,direct \
            --build-arg GOSUMDB=sum.golang.org \
            --build-arg GOLANG_IMAGE=golang:1.27.3-alpine \
            --build-arg NODE_OPTIONS=--max-old-space-size=1536 \
            --build-arg CI_INLINE_TYPECHECK=0 \
            --build-arg GOMEMLIMIT=2200MiB \
            --build-arg GOFLAGS=-p=1 \
            --output "type=docker,dest=${WORKSPACE}/sub2api-image.tar" \
            -f Dockerfile .
          test -s "${WORKSPACE}/sub2api-image.tar"
          du -h "${WORKSPACE}/sub2api-image.tar"
        '''
      }
    }

    stage('投递到 lumio-kin') {
      steps {
        sh label: 'ship', script: '''#!/usr/bin/env bash
          set -euo pipefail
          out=$(ssh -p "${KIN_PORT}" -i "${KIN_KEY}" \
                 -o StrictHostKeyChecking=yes -o IdentitiesOnly=yes -o ConnectTimeout=15 \
                 "root@${KIN_HOST}" load < "${WORKSPACE}/sub2api-image.tar")
          printf '%s\n' "$out"
          grep -q "Loaded image: ${IMAGE_ROOT}:sha-${SHA}" <<<"$out"
        '''
      }
    }
  }

  post {
    always {
      sh 'rm -rf "${WORKSPACE}/src" "${WORKSPACE}/sub2api-image.tar" "${WORKSPACE}/sha.txt" || true'
    }
    success {
      lumioNotify(event: 'build-success', environment: params.STACK, version: env.SHA)
    }
    failure {
      lumioNotify(event: 'build-failure', environment: params.STACK, version: env.SHA,
                  stage: '构建', detail: '查看 Jenkins 构建日志')
    }
  }
}
