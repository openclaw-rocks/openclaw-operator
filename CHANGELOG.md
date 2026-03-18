# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.22.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.21.2...v0.22.0) (2026-03-18)


### Features

* add declarative plugin installation via spec.plugins ([#383](https://github.com/openclaw-rocks/k8s-operator/issues/383)) ([c327bae](https://github.com/openclaw-rocks/k8s-operator/commit/c327baea305c6757aab040e0ac12c8ddb9ebedff)), closes [#372](https://github.com/openclaw-rocks/k8s-operator/issues/372)

## [0.21.2](https://github.com/openclaw-rocks/k8s-operator/compare/v0.21.1...v0.21.2) (2026-03-18)


### Bug Fixes

* inject gateway.handshakeTimeoutMs for K8s startup overhead ([#377](https://github.com/openclaw-rocks/k8s-operator/issues/377)) ([b8e7beb](https://github.com/openclaw-rocks/k8s-operator/commit/b8e7beb810a63911322586f158f898e834de03c7))

## [0.21.1](https://github.com/openclaw-rocks/k8s-operator/compare/v0.21.0...v0.21.1) (2026-03-17)


### Bug Fixes

* drop read-only PVC mount in backup CronJob to fix fsGroup ([#369](https://github.com/openclaw-rocks/k8s-operator/issues/369)) ([29117d3](https://github.com/openclaw-rocks/k8s-operator/commit/29117d313bce55e589d2ef09cec0ee7736cee565))

## [0.21.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.20.1...v0.21.0) (2026-03-17)


### Features

* add optional gateway proxy sidecar (spec.gateway.enabled) ([#339](https://github.com/openclaw-rocks/k8s-operator/issues/339)) ([24ea26c](https://github.com/openclaw-rocks/k8s-operator/commit/24ea26cdb09a04e125c325e13a6b688555252ad2))


### Bug Fixes

* replace invalid metrics config injection with OTel Collector sidecar ([#374](https://github.com/openclaw-rocks/k8s-operator/issues/374)) ([706c77f](https://github.com/openclaw-rocks/k8s-operator/commit/706c77f98a3c06d9023838e279d13efa6dacf87b))

## [0.20.1](https://github.com/openclaw-rocks/k8s-operator/compare/v0.20.0...v0.20.1) (2026-03-16)


### Bug Fixes

* **chromium:** return static /json/version to prevent Playwright bypass ([#365](https://github.com/openclaw-rocks/k8s-operator/issues/365)) ([b21e99e](https://github.com/openclaw-rocks/k8s-operator/commit/b21e99eb9ceecd8277eefd04c0edb833b34e33c8)), closes [#360](https://github.com/openclaw-rocks/k8s-operator/issues/360)
* inject diagnostics.metrics config into OpenClaw when metrics enabled ([a7d4265](https://github.com/openclaw-rocks/k8s-operator/commit/a7d426590d0d948d9c00875d785bce5833c548a9))
* inject diagnostics.metrics config into OpenClaw when metrics enabled ([2db10e3](https://github.com/openclaw-rocks/k8s-operator/commit/2db10e3619c8aeaf6d92dac56f0543b3406ecf9a)), closes [#356](https://github.com/openclaw-rocks/k8s-operator/issues/356)

## [0.20.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.19.1...v0.20.0) (2026-03-16)


### Features

* support spec.availability.runtimeClassName ([62b9ff4](https://github.com/openclaw-rocks/k8s-operator/commit/62b9ff4ed088d4506eda832a182c7f05d8c3320a))
* support spec.availability.runtimeClassName for alternative container runtimes ([79637e9](https://github.com/openclaw-rocks/k8s-operator/commit/79637e9517b7e0be63c44495144ba1e4199a8f00)), closes [#358](https://github.com/openclaw-rocks/k8s-operator/issues/358)


### Bug Fixes

* **chromium:** swap ports so CDP proxy owns 9222, eliminating headless bypass ([6658b33](https://github.com/openclaw-rocks/k8s-operator/commit/6658b33260522f42c4d8ab266d24535164991373))
* **chromium:** swap ports so CDP proxy owns 9222, eliminating headless bypass ([6f42b77](https://github.com/openclaw-rocks/k8s-operator/commit/6f42b7792b13c0b2e42d93d976cf223143da9688)), closes [#270](https://github.com/openclaw-rocks/k8s-operator/issues/270)

## [0.19.1](https://github.com/openclaw-rocks/k8s-operator/compare/v0.19.0...v0.19.1) (2026-03-15)


### Bug Fixes

* add missing --s3-region flag to periodic backup CronJob ([a5cf0ca](https://github.com/openclaw-rocks/k8s-operator/commit/a5cf0ca6ad9612ec34b60f3a4400bca1ad98557a))
* add missing --s3-region flag to periodic backup CronJob ([ff73fd7](https://github.com/openclaw-rocks/k8s-operator/commit/ff73fd7122d7bb52a1646bda570abb9ffdc53889)), closes [#351](https://github.com/openclaw-rocks/k8s-operator/issues/351)

## [0.19.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.18.1...v0.19.0) (2026-03-15)


### Features

* add Service and ServiceMonitor for operator metrics ([4ebc545](https://github.com/openclaw-rocks/k8s-operator/commit/4ebc5450a959877fee37655ef904d3fd0a763362)), closes [#348](https://github.com/openclaw-rocks/k8s-operator/issues/348)

## [0.18.1](https://github.com/openclaw-rocks/k8s-operator/compare/v0.18.0...v0.18.1) (2026-03-14)


### Bug Fixes

* **chromium:** route CDP traffic through proxy instead of bypassing it ([795920b](https://github.com/openclaw-rocks/k8s-operator/commit/795920ba08dcf4f3856bcf86d783bbb99047d9a5))
* **chromium:** route CDP traffic through proxy instead of bypassing it ([c8b3a01](https://github.com/openclaw-rocks/k8s-operator/commit/c8b3a010b2954889f7722b82404610772c555c5b)), closes [#270](https://github.com/openclaw-rocks/k8s-operator/issues/270)
* increase main container startup probe timeout to 300s ([1aad438](https://github.com/openclaw-rocks/k8s-operator/commit/1aad4389a1cb13f43522e3238606417be7b6b5aa))
* increase main container startup probe timeout to 300s ([59c81cd](https://github.com/openclaw-rocks/k8s-operator/commit/59c81cdcc72cdb0bb096fd222a77f64940f44ebd)), closes [#344](https://github.com/openclaw-rocks/k8s-operator/issues/344)

## [0.18.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.17.0...v0.18.0) (2026-03-14)


### Features

* incremental periodic backups with retention-based cleanup ([1f486b6](https://github.com/openclaw-rocks/k8s-operator/commit/1f486b62a230869d691ff5be6d3ac25f4984526c))
* incremental periodic backups with retention-based cleanup ([b1060f3](https://github.com/openclaw-rocks/k8s-operator/commit/b1060f3543e6dcec8e7fa8cbc8255f191a5ba52c))


### Bug Fixes

* include metrics port in NetworkPolicy ingress rules ([be89d79](https://github.com/openclaw-rocks/k8s-operator/commit/be89d79616c20469a0093633db7cca69efae9c8f))
* include metrics port in NetworkPolicy ingress rules ([027ad39](https://github.com/openclaw-rocks/k8s-operator/commit/027ad39f14a40001d9db0ebbbc5255172c1cd836)), closes [#341](https://github.com/openclaw-rocks/k8s-operator/issues/341)

## [0.17.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.16.2...v0.17.0) (2026-03-13)


### Features

* use VolumeClaimTemplates for per-replica PVCs when HPA is enabled ([2bd17dc](https://github.com/openclaw-rocks/k8s-operator/commit/2bd17dc8e2fabd492307b9b9790ee73972d1adec))
* use VolumeClaimTemplates for per-replica PVCs when HPA is enabled ([2969f64](https://github.com/openclaw-rocks/k8s-operator/commit/2969f6448bfe5585b7d0e7f8dbddeebfbc97be45))


### Bug Fixes

* compare VCT specs in VolumeClaimTemplatesEqual to detect size/storageClass changes ([8547a6e](https://github.com/openclaw-rocks/k8s-operator/commit/8547a6ea601d0eb5e26843628f0837088a7083bf))
* emit warning event for orphaned standalone PVC when HPA is enabled ([5094c0a](https://github.com/openclaw-rocks/k8s-operator/commit/5094c0a95ea926318b3fbe4e5142632bd1235421))
* handle immutable VolumeClaimTemplates by recreating StatefulSet on VCT changes ([4fe0fbe](https://github.com/openclaw-rocks/k8s-operator/commit/4fe0fbee80f632b76b419a6cccb16ed05ae516fe))
* normalize VolumeMode on VCTs to prevent reconcile spec drift ([639c3ec](https://github.com/openclaw-rocks/k8s-operator/commit/639c3ec6fd09c86c78173ea5ad478e4eb6b1723b))
* reduce noisy API calls and events in HPA reconcile path ([4c79955](https://github.com/openclaw-rocks/k8s-operator/commit/4c79955b7acadfa84617ee5341d3005eae49e6f2))
* use apiequality.Semantic.DeepEqual for VCT spec comparison ([cb46bc2](https://github.com/openclaw-rocks/k8s-operator/commit/cb46bc2c67d31ec350493d7aeb20b8de48f6b2e3))
* use PascalCase+verb convention for StorageReady condition reason ([baeb26f](https://github.com/openclaw-rocks/k8s-operator/commit/baeb26fc832449c4d892513891935568d76cfd33))
* warn when existingClaim is ignored due to HPA-managed VolumeClaimTemplates ([df28db4](https://github.com/openclaw-rocks/k8s-operator/commit/df28db47a58d44323fc2edf5a451f132836589e2))


### Refactoring

* compute desired StatefulSet once in reconcileStatefulSet ([8c6c779](https://github.com/openclaw-rocks/k8s-operator/commit/8c6c779c61e18332a1d438de2fe91026951f52c9))
* extract IsPersistenceEnabled helper to deduplicate persistence checks ([46e940b](https://github.com/openclaw-rocks/k8s-operator/commit/46e940beae21037593e717d3d7e3943112dda3ff))
* hoist gwSecretName computation to avoid duplication in reconcileStatefulSet ([fba54ae](https://github.com/openclaw-rocks/k8s-operator/commit/fba54aedf501df95e2a7511e0ed2872d6b3100d0))

## [0.16.2](https://github.com/openclaw-rocks/k8s-operator/compare/v0.16.1...v0.16.2) (2026-03-13)


### Bug Fixes

* add npm skill bin path to container PATH ([9237c68](https://github.com/openclaw-rocks/k8s-operator/commit/9237c6846c31cce485b7f38762b4e778580cf99c))
* add npm skill binaries to PATH via global install ([#335](https://github.com/openclaw-rocks/k8s-operator/issues/335)) ([926e034](https://github.com/openclaw-rocks/k8s-operator/commit/926e034edcc43cb729f21fc83df8747c1d8f89b7))
* update e2e test to expect npm install -g ([02944f0](https://github.com/openclaw-rocks/k8s-operator/commit/02944f0a66edb1a33a0feec426b365ef1de81823))

## [0.16.1](https://github.com/openclaw-rocks/k8s-operator/compare/v0.16.0...v0.16.1) (2026-03-13)


### Bug Fixes

* Add get and watch RBAC permissions for pods ([69c6c8b](https://github.com/openclaw-rocks/k8s-operator/commit/69c6c8b0d37bd4640811c85e7cbc7147eb9c4267))
* add get and watch verbs for pods RBAC permission ([ad04174](https://github.com/openclaw-rocks/k8s-operator/commit/ad04174db42b4dbf358a403ff8cdf3ce9dc50c08))
* **chromium:** inject attachOnly, remoteCdpTimeoutMs, and resolved cdpUrl ([8bc75f3](https://github.com/openclaw-rocks/k8s-operator/commit/8bc75f34cdfca024987b67fd94c7d3534f393329)), closes [#270](https://github.com/openclaw-rocks/k8s-operator/issues/270)
* **chromium:** inject attachOnly, timeout, and resolved cdpUrl ([3019c6d](https://github.com/openclaw-rocks/k8s-operator/commit/3019c6d5853282a84b725a2c43a91084f611e5b4))

## [0.16.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.15.1...v0.16.0) (2026-03-12)


### Features

* add PodAnnotations field to pod template ([d7c9fdd](https://github.com/openclaw-rocks/k8s-operator/commit/d7c9fdd69efe8dd5317e33789a0201a9d239015b))
* add PodAnnotations field to pod template ([2ecd1f0](https://github.com/openclaw-rocks/k8s-operator/commit/2ecd1f0c5958590147a78e8f1a2b23af0abdda24))

## [0.15.1](https://github.com/openclaw-rocks/k8s-operator/compare/v0.15.0...v0.15.1) (2026-03-12)


### Bug Fixes

* **backup:** use secretKeyRef instead of plaintext credentials in Job specs ([e45ef9c](https://github.com/openclaw-rocks/k8s-operator/commit/e45ef9c87f5b03463ca7811d55019be74ce02c53))
* **backup:** use secretKeyRef instead of plaintext credentials in Job specs ([e4f2f4d](https://github.com/openclaw-rocks/k8s-operator/commit/e4f2f4d1bb27fb889d594f7f8dbaff3d6e707328)), closes [#322](https://github.com/openclaw-rocks/k8s-operator/issues/322)
* resolve variable shadowing lint errors in mirror secret calls ([a673e60](https://github.com/openclaw-rocks/k8s-operator/commit/a673e605bda6767d9929ba811ac94a525d0685e7))

## [0.15.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.14.1...v0.15.0) (2026-03-12)


### Features

* **backup:** add S3_PROVIDER for multi-cloud workload identity support ([93e22b5](https://github.com/openclaw-rocks/k8s-operator/commit/93e22b50f49dc4df4744df6a369e53bbc1c149aa))
* **backup:** support IRSA and Pod Identity for S3 backup credentials ([8a06d85](https://github.com/openclaw-rocks/k8s-operator/commit/8a06d850965521c60f26bf7d02c7228927e70b2e)), closes [#320](https://github.com/openclaw-rocks/k8s-operator/issues/320)
* **backup:** support workload identity and configurable S3 provider for backup credentials ([4b8e3ca](https://github.com/openclaw-rocks/k8s-operator/commit/4b8e3ca40490da53d990e4dc0fb35e44059cc62b))


### Bug Fixes

* **backup:** validate partial S3 credentials configuration ([e89c4a3](https://github.com/openclaw-rocks/k8s-operator/commit/e89c4a349c44c928738086933012606790484363))

## [0.14.1](https://github.com/openclaw-rocks/k8s-operator/compare/v0.14.0...v0.14.1) (2026-03-11)


### Bug Fixes

* **autoupdate:** scale StatefulSet back up after pre-update backup ([2f52f73](https://github.com/openclaw-rocks/k8s-operator/commit/2f52f735a5205de84a2cb3189ad19b6900eb0829))
* **autoupdate:** scale StatefulSet back up after pre-update backup ([083d752](https://github.com/openclaw-rocks/k8s-operator/commit/083d752e20cc9acaa631b9634e62336afaa155af)), closes [#299](https://github.com/openclaw-rocks/k8s-operator/issues/299)

## [0.14.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.13.0...v0.14.0) (2026-03-11)


### Features

* inject BOOTSTRAP.md for first-run agent onboarding ([bd58ad9](https://github.com/openclaw-rocks/k8s-operator/commit/bd58ad9205023fbafb9d6040812a252443b9671c))

## [0.13.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.12.0...v0.13.0) (2026-03-11)


### Features

* **init-skills:** propagate spec.env and spec.envFrom to skills init container ([d3a08f3](https://github.com/openclaw-rocks/k8s-operator/commit/d3a08f3a5ea5412c3f9eab9e3b009908e7632d2f))
* **init-skills:** propagate spec.env and spec.envFrom to skills init container ([44aff0c](https://github.com/openclaw-rocks/k8s-operator/commit/44aff0cc1be6a29d2195f5f03636ee89066c6e15)), closes [#307](https://github.com/openclaw-rocks/k8s-operator/issues/307)


### Bug Fixes

* **chromium:** use rewrite + bare proxy_pass in named location ([53f23d9](https://github.com/openclaw-rocks/k8s-operator/commit/53f23d9df68eb17f9c2380c935128e6e40b49ba5))
* **chromium:** use rewrite + bare proxy_pass in named location ([#270](https://github.com/openclaw-rocks/k8s-operator/issues/270)) ([f4f0a58](https://github.com/openclaw-rocks/k8s-operator/commit/f4f0a582f3fe1a465e758cd59abebbe895d148ff))
* combine consecutive appends to satisfy gocritic lint ([da357de](https://github.com/openclaw-rocks/k8s-operator/commit/da357de16fbdc06686d0f216fe5dd56d1a38b1ef))
* **skills:** persist ClawHub-installed skills on PVC ([b17bad4](https://github.com/openclaw-rocks/k8s-operator/commit/b17bad47006a9c6b4eab76b49fe2c94f154dab83))
* **skills:** persist ClawHub-installed skills on PVC ([#313](https://github.com/openclaw-rocks/k8s-operator/issues/313)) ([7a7dbfd](https://github.com/openclaw-rocks/k8s-operator/commit/7a7dbfdf83c5a26d9d34af94e6807eee05c8f6e0))
* **web-terminal:** pass -W flag when ReadOnly is false ([a7f806b](https://github.com/openclaw-rocks/k8s-operator/commit/a7f806b11cdf6b96a6b8249a01631ed61c48a986))
* **web-terminal:** pass -W flag when ReadOnly is false ([ff87cfd](https://github.com/openclaw-rocks/k8s-operator/commit/ff87cfd85108982e8c678602866670e20607631a))

## [0.12.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.11.2...v0.12.0) (2026-03-10)


### Features

* apply registry override to init container images ([34aca73](https://github.com/openclaw-rocks/k8s-operator/commit/34aca73a55c773f854c476ae3c1d5c5f73f2a6a4))
* **operator:** add global container image registry override field ([589ddf4](https://github.com/openclaw-rocks/k8s-operator/commit/589ddf44a207b8a66c4c2d10e030dc528275922b))
* **operator:** add global container image registry override field ([3dfa1d0](https://github.com/openclaw-rocks/k8s-operator/commit/3dfa1d0304432d06bc9f8fe861ad49b023353628))


### Bug Fixes

* **chromium:** route WebSocket connections to /chromium endpoint for launch args ([c39bc45](https://github.com/openclaw-rocks/k8s-operator/commit/c39bc45d82121ce8f375cea91a07c87391056730)), closes [#270](https://github.com/openclaw-rocks/k8s-operator/issues/270)
* **chromium:** route WebSocket to /chromium endpoint for launch args ([0e39b89](https://github.com/openclaw-rocks/k8s-operator/commit/0e39b892d56e0764a24f74bbda6cf13b965e0212))
* **resources:** handle trailing slash in registry override. ([6ba4b94](https://github.com/openclaw-rocks/k8s-operator/commit/6ba4b94835cb543b21bfb34ea82cbe3698059ddd))

## [0.11.2](https://github.com/openclaw-rocks/k8s-operator/compare/v0.11.1...v0.11.2) (2026-03-10)


### Bug Fixes

* normalize ClawHub skill slugs and fix documentation ([f738e67](https://github.com/openclaw-rocks/k8s-operator/commit/f738e678893ca233d686f08b733ddf5e2b7fab8d))
* normalize ClawHub skill slugs and fix documentation format ([ab4af55](https://github.com/openclaw-rocks/k8s-operator/commit/ab4af552427d87d9f1f521572b1986959eac475f)), closes [#288](https://github.com/openclaw-rocks/k8s-operator/issues/288)
* update E2E test to expect normalized skill slug ([7f71bc9](https://github.com/openclaw-rocks/k8s-operator/commit/7f71bc9712a17fdba31efc268dc15fdbac449bf4))

## [0.11.1](https://github.com/openclaw-rocks/k8s-operator/compare/v0.11.0...v0.11.1) (2026-03-10)


### Bug Fixes

* redirect nginx http temp dirs to /tmp for read-only rootfs ([#295](https://github.com/openclaw-rocks/k8s-operator/issues/295)) ([ef98bc9](https://github.com/openclaw-rocks/k8s-operator/commit/ef98bc93ec3b3d43a83148d32a4b83476a674ef8))

## [0.11.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.30...v0.11.0) (2026-03-10)


### Features

* add chromium CDP proxy to inject anti-bot Chrome launch args ([1e30d22](https://github.com/openclaw-rocks/k8s-operator/commit/1e30d2243fc0efc01de9fbd7281cae3cc3e1bd8f)), closes [#270](https://github.com/openclaw-rocks/k8s-operator/issues/270)
* chromium CDP proxy for anti-bot launch args ([e7d9d86](https://github.com/openclaw-rocks/k8s-operator/commit/e7d9d86d97035cb8bcd4e4331a24f9b9c3c23951))
* **resources:** add logging and validation for resource quantities ([0dc3508](https://github.com/openclaw-rocks/k8s-operator/commit/0dc35087c96bebb78c720b9dd803e77e8e64340a))
* validate existing PVC and improve resource parsing ([351c87a](https://github.com/openclaw-rocks/k8s-operator/commit/351c87a7b07ebe86338631d7018347ccfee10b2d))


### Bug Fixes

* handle merge commits in release tag creation step ([370edf6](https://github.com/openclaw-rocks/k8s-operator/commit/370edf6e2459147e06d6cf69d5e4f225f0e98da0))
* handle merge commits in release tag creation step ([07b45bf](https://github.com/openclaw-rocks/k8s-operator/commit/07b45bfb2d1e2ca321e760921862542861c8974e))
* resolve chromium sidecar startup race and NetworkPolicy gaps ([8579946](https://github.com/openclaw-rocks/k8s-operator/commit/857994691db3206fd904732f03d6fa677a809253))
* resolve chromium sidecar startup race and NetworkPolicy gaps ([8e7fc99](https://github.com/openclaw-rocks/k8s-operator/commit/8e7fc99f6dc4cbedf264587547c07411534eefa5)), closes [#270](https://github.com/openclaw-rocks/k8s-operator/issues/270)

## [0.10.30](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.29...v0.10.30) (2026-03-10)


### Features

* idempotent ClawHub skill installs for persistent storage ([b02c5f2](https://github.com/openclaw-rocks/k8s-operator/commit/b02c5f244315759afb0dfd55f2fd78754af7cd5b))


### Bug Fixes

* add activeDeadlineSeconds and startingDeadlineSeconds to backup CronJob ([d5a5a0a](https://github.com/openclaw-rocks/k8s-operator/commit/d5a5a0ad27a6928dce9a5118a89b785df64cfe17)), closes [#286](https://github.com/openclaw-rocks/k8s-operator/issues/286)
* add deadline safeguards to backup CronJob ([5f3715d](https://github.com/openclaw-rocks/k8s-operator/commit/5f3715d77ecdef97f998b18a1ecabb4d15d9a680))
* add K8s API port 6443 egress when tailscale is enabled ([5968cc1](https://github.com/openclaw-rocks/k8s-operator/commit/5968cc1cd28706cc811ed695cdeefacd5c6e2784))
* inject POD_NAMESPACE env via Downward API ([b793c0f](https://github.com/openclaw-rocks/k8s-operator/commit/b793c0ff352abd1182a7e7d1847f3105a02aad94))
* inject POD_NAMESPACE env via Downward API in operator deployment ([8b5ae53](https://github.com/openclaw-rocks/k8s-operator/commit/8b5ae53605cc1a79981ddeb9d95973bc3b5c92cd)), closes [#281](https://github.com/openclaw-rocks/k8s-operator/issues/281)

## [0.10.29](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.28...v0.10.29) (2026-03-09)


### Bug Fixes

* increase chromium startup probe timeout from 2s to 5s ([#279](https://github.com/openclaw-rocks/k8s-operator/issues/279)) ([2ccf3ec](https://github.com/openclaw-rocks/k8s-operator/commit/2ccf3ec252daefa846e9c7c027dd40f870b996a6)), closes [#270](https://github.com/openclaw-rocks/k8s-operator/issues/270)

## [0.10.28](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.27...v0.10.28) (2026-03-09)


### Bug Fixes

* auto-inject 127.0.0.0/8 into gateway.trustedProxies ([#276](https://github.com/openclaw-rocks/k8s-operator/issues/276)) ([e7ecc5c](https://github.com/openclaw-rocks/k8s-operator/commit/e7ecc5c6f1c50b1b6f26621b9e99d22266dadd34)), closes [#274](https://github.com/openclaw-rocks/k8s-operator/issues/274)
* handle OCI pagination in registry tag resolver ([#275](https://github.com/openclaw-rocks/k8s-operator/issues/275)) ([2fcf3dd](https://github.com/openclaw-rocks/k8s-operator/commit/2fcf3dd41e2e3c24a440fed155019ae664b36255))

## [0.10.27](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.26...v0.10.27) (2026-03-09)


### Features

* support persistent Chromium browser profiles via PVC ([#271](https://github.com/openclaw-rocks/k8s-operator/issues/271)) ([9d80414](https://github.com/openclaw-rocks/k8s-operator/commit/9d804148da28ad4ab1640d1da15d7b100ade4347)), closes [#267](https://github.com/openclaw-rocks/k8s-operator/issues/267)

## [0.10.26](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.25...v0.10.26) (2026-03-09)


### Features

* persist Tailscale state across pod restarts via TS_KUBE_SECRET ([#265](https://github.com/openclaw-rocks/k8s-operator/issues/265)) ([0a9601d](https://github.com/openclaw-rocks/k8s-operator/commit/0a9601d06972f6d507705c350987d1c12d6c5759)), closes [#262](https://github.com/openclaw-rocks/k8s-operator/issues/262)


### Bug Fixes

* remove invalid llmConfig from webhook validation and docs ([#261](https://github.com/openclaw-rocks/k8s-operator/issues/261)) ([e8f7399](https://github.com/openclaw-rocks/k8s-operator/commit/e8f739940e92d2a0a5b2b27ee7cadcab07df6026))
* respect pod-level runAsNonRoot in container security contexts ([#266](https://github.com/openclaw-rocks/k8s-operator/issues/266)) ([ad21b4c](https://github.com/openclaw-rocks/k8s-operator/commit/ad21b4c59260117ed9d2f13ba8278fb4fe7e56d1)), closes [#263](https://github.com/openclaw-rocks/k8s-operator/issues/263)

## [0.10.25](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.24...v0.10.25) (2026-03-07)


### Features

* enable writable package installs on read-only root filesystem ([#254](https://github.com/openclaw-rocks/k8s-operator/issues/254)) ([8d5f4ba](https://github.com/openclaw-rocks/k8s-operator/commit/8d5f4ba3795980589a566cf2866d3af2bf588987))

## [0.10.24](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.23...v0.10.24) (2026-03-06)


### Bug Fixes

* use service DNS for Chromium CDP URL instead of localhost ([#252](https://github.com/openclaw-rocks/k8s-operator/issues/252)) ([70b9ec4](https://github.com/openclaw-rocks/k8s-operator/commit/70b9ec48c156cc20d9c9eaf6837e39350f232a40))

## [0.10.23](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.22...v0.10.23) (2026-03-06)


### Bug Fixes

* correct attachOnly field placement in browser configuration ([#250](https://github.com/openclaw-rocks/k8s-operator/issues/250)) ([04d44af](https://github.com/openclaw-rocks/k8s-operator/commit/04d44af9ff6a0ac9792e9746e23ed2b6d028aa4d))

## [0.10.22](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.21...v0.10.22) (2026-03-05)


### Features

* inject default anti-bot-detection flags for Chromium sidecar ([#247](https://github.com/openclaw-rocks/k8s-operator/issues/247)) ([4a38b4d](https://github.com/openclaw-rocks/k8s-operator/commit/4a38b4dd7a8b73108448d189561dd565bf10a633))

## [0.10.21](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.20...v0.10.21) (2026-03-05)


### Bug Fixes

* disable device auth for Control UI in K8s environments ([#238](https://github.com/openclaw-rocks/k8s-operator/issues/238)) ([c5c420c](https://github.com/openclaw-rocks/k8s-operator/commit/c5c420c40d0f01c58e49d5d535f60f5db4b0651d)), closes [#233](https://github.com/openclaw-rocks/k8s-operator/issues/233)
* propagate nodeSelector and tolerations to backup CronJob pods ([#245](https://github.com/openclaw-rocks/k8s-operator/issues/245)) ([98ef456](https://github.com/openclaw-rocks/k8s-operator/commit/98ef4568d4156f5ce27c9d253dd3da6bf9350c12)), closes [#244](https://github.com/openclaw-rocks/k8s-operator/issues/244)
* use localhost for Chromium CDP URL to support IPv6 clusters ([#243](https://github.com/openclaw-rocks/k8s-operator/issues/243)) ([08dc2c2](https://github.com/openclaw-rocks/k8s-operator/commit/08dc2c27ce98dc0b54b18a295ec0823d74b90d44)), closes [#228](https://github.com/openclaw-rocks/k8s-operator/issues/228)

## [0.10.20](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.19...v0.10.20) (2026-03-04)


### Features

* add attachOnly and disable device auth in browser profiles ([#236](https://github.com/openclaw-rocks/k8s-operator/issues/236)) ([44dd9ce](https://github.com/openclaw-rocks/k8s-operator/commit/44dd9cee63c5a442f0dca2e6c1144224489b7ff3))

## [0.10.19](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.18...v0.10.19) (2026-03-04)


### Features

* auto-inject gateway.controlUi.allowedOrigins ([#234](https://github.com/openclaw-rocks/k8s-operator/issues/234)) ([#235](https://github.com/openclaw-rocks/k8s-operator/issues/235)) ([46b5445](https://github.com/openclaw-rocks/k8s-operator/commit/46b5445e299e4af252568fcb043bb390373ca12c))

## [0.10.18](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.17...v0.10.18) (2026-03-03)


### Features

* switch probes from exec to httpGet ([#231](https://github.com/openclaw-rocks/k8s-operator/issues/231)) ([d5b7754](https://github.com/openclaw-rocks/k8s-operator/commit/d5b7754400e3b1d3e41c57237b03aefc8bc76525))

## [0.10.17](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.16...v0.10.17) (2026-03-02)


### Bug Fixes

* add configurable timeout for BackingUp phase to prevent stuck instances ([#226](https://github.com/openclaw-rocks/k8s-operator/issues/226)) ([778a642](https://github.com/openclaw-rocks/k8s-operator/commit/778a6426d4f2b26845f110887b3bc41f02ceb0ac)), closes [#224](https://github.com/openclaw-rocks/k8s-operator/issues/224)
* make skill pack resolution non-blocking to prevent provisioning failures ([#227](https://github.com/openclaw-rocks/k8s-operator/issues/227)) ([ffb2485](https://github.com/openclaw-rocks/k8s-operator/commit/ffb24852190e66844c9fe0f69ba28db8c9fade8f)), closes [#225](https://github.com/openclaw-rocks/k8s-operator/issues/225)

## [0.10.16](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.15...v0.10.16) (2026-03-01)


### Bug Fixes

* propagate nodeSelector and tolerations to backup/restore Jobs ([#221](https://github.com/openclaw-rocks/k8s-operator/issues/221)) ([342c7ae](https://github.com/openclaw-rocks/k8s-operator/commit/342c7ae032cad88b5f4e15009c2accca8fd1ddb6))

## [0.10.15](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.14...v0.10.15) (2026-02-27)


### Features

* GitHub-based skill pack resolution ([#218](https://github.com/openclaw-rocks/k8s-operator/issues/218)) ([077dfa6](https://github.com/openclaw-rocks/k8s-operator/commit/077dfa6b52ec71c312a1e9338788b331d2cfd27a))

## [0.10.14](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.13...v0.10.14) (2026-02-27)


### Bug Fixes

* prevent StatefulSet reconciliation loop from server-side defaults ([#217](https://github.com/openclaw-rocks/k8s-operator/issues/217)) ([0617b46](https://github.com/openclaw-rocks/k8s-operator/commit/0617b46147cf6221ece0020b613ec8364ec8bc53))

## [0.10.13](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.12...v0.10.13) (2026-02-27)


### Bug Fixes

* use Node.js TCP connect for health probes instead of wget ([#215](https://github.com/openclaw-rocks/k8s-operator/issues/215)) ([ecb7474](https://github.com/openclaw-rocks/k8s-operator/commit/ecb7474c2ea40775b3a5df5f008b299d319f0d75))

## [0.10.12](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.11...v0.10.12) (2026-02-26)


### Features

* add periodic scheduled backups via CronJob ([#207](https://github.com/openclaw-rocks/k8s-operator/issues/207)) ([bf29965](https://github.com/openclaw-rocks/k8s-operator/commit/bf299650b8aad631fdfa6c3427f414bda9b7d511))


### Bug Fixes

* add optional S3_REGION support for MinIO backups ([#212](https://github.com/openclaw-rocks/k8s-operator/issues/212)) ([c5e96c8](https://github.com/openclaw-rocks/k8s-operator/commit/c5e96c89660566d5b35a137d8a098ca18cf51234)), closes [#205](https://github.com/openclaw-rocks/k8s-operator/issues/205)
* **chromium:** pass extraArgs via DEFAULT_LAUNCH_ARGS env instead of container Args ([#211](https://github.com/openclaw-rocks/k8s-operator/issues/211)) ([ec79758](https://github.com/openclaw-rocks/k8s-operator/commit/ec797581aa068ebd48cb55c11de048f41024c826)), closes [#209](https://github.com/openclaw-rocks/k8s-operator/issues/209)

## [0.10.11](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.10...v0.10.11) (2026-02-26)


### Bug Fixes

* store plaintext username and password in auto-generated basic auth secret ([#208](https://github.com/openclaw-rocks/k8s-operator/issues/208)) ([179b4a6](https://github.com/openclaw-rocks/k8s-operator/commit/179b4a60b94cf2e7fae5ddb293c9bfcc534f40ea)), closes [#201](https://github.com/openclaw-rocks/k8s-operator/issues/201)

## [0.10.10](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.9...v0.10.10) (2026-02-26)


### Features

* add performance benchmarks for resource builders ([#197](https://github.com/openclaw-rocks/k8s-operator/issues/197)) ([41efb29](https://github.com/openclaw-rocks/k8s-operator/commit/41efb29b1de83aa74520f06a6d0dd4483eb08dcc))

## [0.10.9](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.8...v0.10.9) (2026-02-26)


### Features

* add Operator SDK scorecard testing ([#198](https://github.com/openclaw-rocks/k8s-operator/issues/198)) ([3b44d9a](https://github.com/openclaw-rocks/k8s-operator/commit/3b44d9a4f4533988be44cca2f809ab70a2c93fc7))
* add topology spread constraints support ([#196](https://github.com/openclaw-rocks/k8s-operator/issues/196)) ([98ba176](https://github.com/openclaw-rocks/k8s-operator/commit/98ba176822bc72e1a27497ab66f122715ac024fc))

## [0.10.8](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.7...v0.10.8) (2026-02-26)


### Features

* **chromium:** add extraArgs and extraEnv fields to ChromiumSpec ([#187](https://github.com/openclaw-rocks/k8s-operator/issues/187)) ([2878482](https://github.com/openclaw-rocks/k8s-operator/commit/2878482029cf0d4f4f4976cacbb594c66bb8a339))
* **chromium:** add extraArgs/extraEnv to ChromiumSpec, fix issues [#189](https://github.com/openclaw-rocks/k8s-operator/issues/189)-[#193](https://github.com/openclaw-rocks/k8s-operator/issues/193) ([#194](https://github.com/openclaw-rocks/k8s-operator/issues/194)) ([cba8d1a](https://github.com/openclaw-rocks/k8s-operator/commit/cba8d1a66c972a434d182fd80e815a44e6e79990))

## [0.10.7](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.6...v0.10.7) (2026-02-25)


### Bug Fixes

* resolve chromium sidecar port conflict and unreachable CDP ([#183](https://github.com/openclaw-rocks/k8s-operator/issues/183)) ([2d3d212](https://github.com/openclaw-rocks/k8s-operator/commit/2d3d2127e6f2ee2566e748e993b5aceae22c80a4)), closes [#180](https://github.com/openclaw-rocks/k8s-operator/issues/180)

## [0.10.6](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.5...v0.10.6) (2026-02-24)


### Bug Fixes

* make Probes a pointer type to accept null/omitted values ([#181](https://github.com/openclaw-rocks/k8s-operator/issues/181)) ([df42069](https://github.com/openclaw-rocks/k8s-operator/commit/df42069191e451a756ced3b74faaed3cf44acab0)), closes [#179](https://github.com/openclaw-rocks/k8s-operator/issues/179)

## [0.10.5](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.4...v0.10.5) (2026-02-24)


### Features

* add Tailscale sidecar for working tailnet integration ([#177](https://github.com/openclaw-rocks/k8s-operator/issues/177)) ([d956925](https://github.com/openclaw-rocks/k8s-operator/commit/d956925bb60437aa67ff82f961228933e286e0fc))

## [0.10.4](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.3...v0.10.4) (2026-02-24)


### Features

* add nginx gateway proxy sidecar for loopback-bound gateway ([#175](https://github.com/openclaw-rocks/k8s-operator/issues/175)) ([a52383b](https://github.com/openclaw-rocks/k8s-operator/commit/a52383b121de80229b1ae7b03622cd10561ac5a5))

## [0.10.3](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.2...v0.10.3) (2026-02-23)


### Bug Fixes

* auto-set gateway.bind=loopback and use exec probes for Tailscale serve/funnel ([#170](https://github.com/openclaw-rocks/k8s-operator/issues/170)) ([e26694b](https://github.com/openclaw-rocks/k8s-operator/commit/e26694b5bd38d7bd4a286f1e6caf5d3438bf6366)), closes [#167](https://github.com/openclaw-rocks/k8s-operator/issues/167)
* expose metrics port in Service, StatefulSet, and ServiceMonitor ([#169](https://github.com/openclaw-rocks/k8s-operator/issues/169)) ([049d097](https://github.com/openclaw-rocks/k8s-operator/commit/049d097d81a42493a224bf129a243162685b1e5d)), closes [#166](https://github.com/openclaw-rocks/k8s-operator/issues/166)
* move CRDs from Helm crds/ to templates/ for upgrade support ([#173](https://github.com/openclaw-rocks/k8s-operator/issues/173)) ([599b394](https://github.com/openclaw-rocks/k8s-operator/commit/599b39450a91a858e15d5ccf5f702db0c2c82392)), closes [#168](https://github.com/openclaw-rocks/k8s-operator/issues/168)

## [0.10.2](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.1...v0.10.2) (2026-02-23)


### Bug Fixes

* break reconciliation tight loop caused by unconditional status writes ([#163](https://github.com/openclaw-rocks/k8s-operator/issues/163)) ([88921b7](https://github.com/openclaw-rocks/k8s-operator/commit/88921b7fafe3e703441030d49885dc3277c61213)), closes [#161](https://github.com/openclaw-rocks/k8s-operator/issues/161)
* use single-quoted node -e argument in merge mode scripts ([#164](https://github.com/openclaw-rocks/k8s-operator/issues/164)) ([a661ed8](https://github.com/openclaw-rocks/k8s-operator/commit/a661ed8d7fbaa91bbedcd9a198dbe74d52c27444)), closes [#162](https://github.com/openclaw-rocks/k8s-operator/issues/162)

## [0.10.1](https://github.com/openclaw-rocks/k8s-operator/compare/v0.10.0...v0.10.1) (2026-02-22)


### Features

* add ttyd web terminal managed sidecar ([#159](https://github.com/openclaw-rocks/k8s-operator/issues/159)) ([a9e1bce](https://github.com/openclaw-rocks/k8s-operator/commit/a9e1bceb9897391cdf75459c6b4beaddce64c201))

## [0.10.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.23...v0.10.0) (2026-02-22)


### ⚠ BREAKING CHANGES

* The backup credentials Secret name changes from `b2-backup-credentials` to `s3-backup-credentials`, and the expected keys change from B2_BUCKET/B2_KEY_ID/B2_APP_KEY/B2_ENDPOINT to S3_BUCKET/S3_ACCESS_KEY_ID/S3_SECRET_ACCESS_KEY/S3_ENDPOINT.

### Features

* rename B2/Backblaze to generic S3-compatible storage ([#157](https://github.com/openclaw-rocks/k8s-operator/issues/157)) ([df76683](https://github.com/openclaw-rocks/k8s-operator/commit/df766839d6fdef920dd85960e46f6a4d2299bc9f))

## [0.9.23](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.22...v0.9.23) (2026-02-22)


### Features

* add HPA auto-scaling and Auto Pilot capability level ([#155](https://github.com/openclaw-rocks/k8s-operator/issues/155)) ([024e51a](https://github.com/openclaw-rocks/k8s-operator/commit/024e51aac294851ae0c9102d89c3ab38e9768d59))

## [0.9.22](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.21...v0.9.22) (2026-02-22)


### Features

* add OpenClawSelfConfig CRD for agent self-modification ([#146](https://github.com/openclaw-rocks/k8s-operator/issues/146)) ([2351737](https://github.com/openclaw-rocks/k8s-operator/commit/235173750618afd32e64c571604c3bf753add2ef))

## [0.9.21](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.20...v0.9.21) (2026-02-21)


### Bug Fixes

* set CSV capabilities to Deep Insights (Level 4) ([#152](https://github.com/openclaw-rocks/k8s-operator/issues/152)) ([1fe72ed](https://github.com/openclaw-rocks/k8s-operator/commit/1fe72ed7369e3cc73255c7a6cde2dfb6ac841b39))

## [0.9.20](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.19...v0.9.20) (2026-02-21)


### Features

* add Level 4 Deep Insights - auto-provisioned PrometheusRule and Grafana dashboards ([#149](https://github.com/openclaw-rocks/k8s-operator/issues/149)) ([3c46765](https://github.com/openclaw-rocks/k8s-operator/commit/3c46765f5e86a24ca904b58be1fce863cfdce4ff))

## [0.9.19](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.18...v0.9.19) (2026-02-21)


### Features

* add custom service ports and ingress backend port support ([#144](https://github.com/openclaw-rocks/k8s-operator/issues/144)) ([#145](https://github.com/openclaw-rocks/k8s-operator/issues/145)) ([d0604c1](https://github.com/openclaw-rocks/k8s-operator/commit/d0604c141889c37fb957c002ef7aeca7cecee10c))

## [0.9.18](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.17...v0.9.18) (2026-02-20)


### Features

* improve OperatorHub and ArtifactHub listing quality ([#142](https://github.com/openclaw-rocks/k8s-operator/issues/142)) ([b15937c](https://github.com/openclaw-rocks/k8s-operator/commit/b15937c87da175cf72f1b7981c6bcef1b8330bfe))

## [0.9.17](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.16...v0.9.17) (2026-02-20)


### Bug Fixes

* configMapRef bypasses gateway auth enrichment ([#138](https://github.com/openclaw-rocks/k8s-operator/issues/138)) ([1322d5e](https://github.com/openclaw-rocks/k8s-operator/commit/1322d5e8e3141a9c5a4ea7298ddc0149d2a1d11c))

## [0.9.16](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.15...v0.9.16) (2026-02-20)


### Features

* support npm packages in skills field and disable lifecycle scripts ([#137](https://github.com/openclaw-rocks/k8s-operator/issues/137)) ([a9db9d0](https://github.com/openclaw-rocks/k8s-operator/commit/a9db9d091a357bca07b260ed6e7f9560917f9e9d)), closes [#131](https://github.com/openclaw-rocks/k8s-operator/issues/131) [#91](https://github.com/openclaw-rocks/k8s-operator/issues/91)

## [0.9.15](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.14...v0.9.15) (2026-02-19)


### Bug Fixes

* prevent StatefulSet spec drift on every reconcile ([#133](https://github.com/openclaw-rocks/k8s-operator/issues/133)) ([b6fa7b3](https://github.com/openclaw-rocks/k8s-operator/commit/b6fa7b3f257e51593fb01aaa42a8e4b89e9de50b))

## [0.9.14](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.13...v0.9.14) (2026-02-19)


### Bug Fixes

* restore config on container restart via postStart hook ([#128](https://github.com/openclaw-rocks/k8s-operator/issues/128)) ([38ea2c5](https://github.com/openclaw-rocks/k8s-operator/commit/38ea2c53adc1a4ba71bf307780b8c415d18fad6d))

## [0.9.13](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.12...v0.9.13) (2026-02-19)


### Bug Fixes

* inject browser config for Chromium sidecar ([#126](https://github.com/openclaw-rocks/k8s-operator/issues/126)) ([570344e](https://github.com/openclaw-rocks/k8s-operator/commit/570344e167702092bd7e5640d7c1aff0340e01e1))
* use copyFileSync instead of renameSync in merge mode init container ([#121](https://github.com/openclaw-rocks/k8s-operator/issues/121)) ([72bd962](https://github.com/openclaw-rocks/k8s-operator/commit/72bd962d492fdfd3ccc525523e3bb6f12de3ac70)), closes [#120](https://github.com/openclaw-rocks/k8s-operator/issues/120)

## [0.9.12](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.11...v0.9.12) (2026-02-18)


### Features

* add first-class Ollama sidecar support ([860225e](https://github.com/openclaw-rocks/k8s-operator/commit/860225e8e656d228218056ed6be8937f897ba582))
* add native Tailscale integration via CRD fields ([#115](https://github.com/openclaw-rocks/k8s-operator/issues/115)) ([c3a2ae4](https://github.com/openclaw-rocks/k8s-operator/commit/c3a2ae49b73836780aebb192fdfe931d445a1751))

## [0.9.11](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.10...v0.9.11) (2026-02-18)


### Bug Fixes

* resolve flaky backup/restore tests and autorelease label API ([#117](https://github.com/openclaw-rocks/k8s-operator/issues/117)) ([da7a2a5](https://github.com/openclaw-rocks/k8s-operator/commit/da7a2a58f0cb82bac091e954179efdd4827d2b78))

## [0.9.10](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.9...v0.9.10) (2026-02-18)


### Bug Fixes

* emit provider-aware ingress annotations based on className ([#109](https://github.com/openclaw-rocks/k8s-operator/issues/109)) ([#110](https://github.com/openclaw-rocks/k8s-operator/issues/110)) ([c040df6](https://github.com/openclaw-rocks/k8s-operator/commit/c040df69f77e95ea8e284e84ad7bf86cd03df1ed))
* graceful deletion when B2 backup credentials are not configured ([#112](https://github.com/openclaw-rocks/k8s-operator/issues/112)) ([10b59be](https://github.com/openclaw-rocks/k8s-operator/commit/10b59be0368d2f4c2ad5069621ecf62609d924eb)), closes [#111](https://github.com/openclaw-rocks/k8s-operator/issues/111)
* use shell-capable images for distroless init containers ([#108](https://github.com/openclaw-rocks/k8s-operator/issues/108)) ([2c87e68](https://github.com/openclaw-rocks/k8s-operator/commit/2c87e68e2e8b6bb94c95fb2ae751084843ecf2af))

## [0.9.9](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.8...v0.9.9) (2026-02-17)


### Bug Fixes

* inject gateway.bind=lan so vanilla deployments pass health probes ([#102](https://github.com/openclaw-rocks/k8s-operator/issues/102)) ([7c63d86](https://github.com/openclaw-rocks/k8s-operator/commit/7c63d862b7e09c1d44368205c915064d4dbe25e1)), closes [#101](https://github.com/openclaw-rocks/k8s-operator/issues/101)

## [0.9.8](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.7...v0.9.8) (2026-02-17)


### Bug Fixes

* sync Helm RBAC and add gateway.existingSecret ([#98](https://github.com/openclaw-rocks/k8s-operator/issues/98)) ([33dbc2c](https://github.com/openclaw-rocks/k8s-operator/commit/33dbc2c15344ef857ec8e5d70f15544f9f5a12a0))

## [0.9.7](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.6...v0.9.7) (2026-02-17)


### Bug Fixes

* use server-side apply for CRD installation and update README ([#94](https://github.com/openclaw-rocks/k8s-operator/issues/94)) ([73b0677](https://github.com/openclaw-rocks/k8s-operator/commit/73b0677d577ada187af7298e0b685cf25161ba12))

## [0.9.6](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.5...v0.9.6) (2026-02-16)


### Features

* add runtime dependency init containers for pnpm and Python/uv ([#89](https://github.com/openclaw-rocks/k8s-operator/issues/89)) ([#90](https://github.com/openclaw-rocks/k8s-operator/issues/90)) ([b6a583c](https://github.com/openclaw-rocks/k8s-operator/commit/b6a583cfa368a49d0054943b5f055e925c2c2e5e))

## [0.9.5](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.4...v0.9.5) (2026-02-16)


### Features

* add Phase 2+3 features and CVE-2025-22868 fix ([#84](https://github.com/openclaw-rocks/k8s-operator/issues/84)) ([94d4273](https://github.com/openclaw-rocks/k8s-operator/commit/94d427300aece52946929d3360e612ebbdba8441))
* add read-only rootfs, config merge mode, skill installation, and secret rotation detection ([#82](https://github.com/openclaw-rocks/k8s-operator/issues/82)) ([abd7911](https://github.com/openclaw-rocks/k8s-operator/commit/abd7911b76c829127c133d251927f5862f3cdeda))
* auto-generate gateway token auth for OpenClaw instances ([#85](https://github.com/openclaw-rocks/k8s-operator/issues/85)) ([6ee7eca](https://github.com/openclaw-rocks/k8s-operator/commit/6ee7eca5002b7035f15e1042a18d6199e50823c1)), closes [#83](https://github.com/openclaw-rocks/k8s-operator/issues/83)

## [0.9.4](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.3...v0.9.4) (2026-02-16)


### Bug Fixes

* auto-label release PRs to prevent release-please stalling ([#80](https://github.com/openclaw-rocks/k8s-operator/issues/80)) ([c1bce0a](https://github.com/openclaw-rocks/k8s-operator/commit/c1bce0add378e1917c086c6c6e7f4779ac7f45af))

## [0.9.3](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.2...v0.9.3) (2026-02-16)


### Features

* add auto-rollback on failed update with health check monitoring ([#75](https://github.com/openclaw-rocks/k8s-operator/issues/75)) ([1dff347](https://github.com/openclaw-rocks/k8s-operator/commit/1dff347ce46c8608be6a92640b9a5bf1bee53f28))

## [0.9.2](https://github.com/openclaw-rocks/k8s-operator/compare/v0.9.1...v0.9.2) (2026-02-16)


### Features

* add opt-in auto-update for OCI registry version tracking ([#71](https://github.com/openclaw-rocks/k8s-operator/issues/71)) ([5ce624e](https://github.com/openclaw-rocks/k8s-operator/commit/5ce624e72e70a185700fced9b4cc6c2461d74b91))
* add webhook warning for latest image tag ([#67](https://github.com/openclaw-rocks/k8s-operator/issues/67)) ([6791624](https://github.com/openclaw-rocks/k8s-operator/commit/6791624d8ec5e46fc95b02f125ac38b1578a30c7))


### Bug Fixes

* add skip-backup annotation to E2E test instance ([#69](https://github.com/openclaw-rocks/k8s-operator/issues/69)) ([33ab056](https://github.com/openclaw-rocks/k8s-operator/commit/33ab0563ebfdfb4817a8cd4627400af35ae3a699))
* extract imageTagLatest constant to satisfy goconst linter ([#68](https://github.com/openclaw-rocks/k8s-operator/issues/68)) ([fc14d1a](https://github.com/openclaw-rocks/k8s-operator/commit/fc14d1a12406373cf0fd5a6cf779e64594577841))
* suppress gosec false positive and handle existing releases ([aef5468](https://github.com/openclaw-rocks/k8s-operator/commit/aef5468b57b3e787b9dbff82470362f5918d4bd0))
* suppress gosec G101 false positive and handle existing releases ([182fafb](https://github.com/openclaw-rocks/k8s-operator/commit/182fafb026d7733b4f6c9c5e918366fab71da79c))
* sync release-please manifest to v0.9.1 ([#72](https://github.com/openclaw-rocks/k8s-operator/issues/72)) ([88f50a9](https://github.com/openclaw-rocks/k8s-operator/commit/88f50a9a2492d20922d7dfe2781e1d8c0fcdb1dd))

## [0.6.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.5.0...v0.6.0) (2026-02-13)


### ⚠ BREAKING CHANGES

* CRD API group changed from `openclaw.openclaw.io` to `openclaw.openclaw.rocks`. Existing CRDs must be deleted and re-created. This is acceptable at v1alpha1 stability level.

### Features

* Add nautical banner image with crab captain at Kubernetes helm ([84d1854](https://github.com/openclaw-rocks/k8s-operator/commit/84d1854cb7200587d6745c9ae3dc29c049fc3b8e))
* Add observability, testing, docs, and dev tooling ([5af27e1](https://github.com/openclaw-rocks/k8s-operator/commit/5af27e1884285d385356ee548f32a62dc8608a95))
* Add observability, testing, docs, and dev tooling for production readiness ([a8c063b](https://github.com/openclaw-rocks/k8s-operator/commit/a8c063b1e06143e06eacf84419ecbc31a9435684))
* Add support for custom sidecar containers ([#27](https://github.com/openclaw-rocks/k8s-operator/issues/27)) ([f0071f7](https://github.com/openclaw-rocks/k8s-operator/commit/f0071f72dd92b248223f53b7af9b2f3eb9a184be)), closes [#24](https://github.com/openclaw-rocks/k8s-operator/issues/24)
* Initial OpenClaw Kubernetes Operator implementation ([6d873ff](https://github.com/openclaw-rocks/k8s-operator/commit/6d873ff855273c9e51337181e657ced2b82af711))
* Inject CHROMIUM_URL env var into main container when sidecar is enabled ([8553cc7](https://github.com/openclaw-rocks/k8s-operator/commit/8553cc7ec0b41675bd815a0f8febd0622bbf068c))
* Inject CHROMIUM_URL env var when sidecar is enabled ([b80dabe](https://github.com/openclaw-rocks/k8s-operator/commit/b80dabe32dbf2ff0f755c92f4e83e2467050bd89))
* Replace manual release steps with GoReleaser ([#2](https://github.com/openclaw-rocks/k8s-operator/issues/2)) ([fedc497](https://github.com/openclaw-rocks/k8s-operator/commit/fedc49766d414e0a2d96bc9c080abcb82b2b2202))
* Support custom egress rules in NetworkPolicy ([#15](https://github.com/openclaw-rocks/k8s-operator/issues/15)) ([#16](https://github.com/openclaw-rocks/k8s-operator/issues/16)) ([c62dc09](https://github.com/openclaw-rocks/k8s-operator/commit/c62dc09cec6112cf4625175423d033310ce50dc3))
* update banner with real OpenClaw logo and Kubernetes logo ([#37](https://github.com/openclaw-rocks/k8s-operator/issues/37)) ([840adb9](https://github.com/openclaw-rocks/k8s-operator/commit/840adb9e285c14dffe1bfed17e9b9411b05f4538))


### Bug Fixes

* Add leader election RBAC and E2E test infrastructure ([4eb6019](https://github.com/openclaw-rocks/k8s-operator/commit/4eb601994f3b87bde07f4c28a51f1dd010f045d4))
* Apply same CreateOrUpdate pattern to ServiceMonitor reconciler ([#30](https://github.com/openclaw-rocks/k8s-operator/issues/30)) ([c1aaa36](https://github.com/openclaw-rocks/k8s-operator/commit/c1aaa3639b7cb82eb95696bbe06b98817d46a55e))
* Bump chromium /dev/shm from 256Mi to 1Gi ([925d7b8](https://github.com/openclaw-rocks/k8s-operator/commit/925d7b84c0630b73f581fca32d8da68ca582cd19))
* Bump chromium /dev/shm sizeLimit from 256Mi to 1Gi ([a4b3fbb](https://github.com/openclaw-rocks/k8s-operator/commit/a4b3fbb005fd40a743a54dcb8ce42c0db9ef7b27))
* change CRD API group domain from openclaw.io to openclaw.rocks ([#41](https://github.com/openclaw-rocks/k8s-operator/issues/41)) ([5bae852](https://github.com/openclaw-rocks/k8s-operator/commit/5bae852810d249358fad1453694503298851208c))
* Chromium sidecar crash (UID mismatch + read-only rootfs) and lint ([#14](https://github.com/openclaw-rocks/k8s-operator/issues/14)) ([febe1d3](https://github.com/openclaw-rocks/k8s-operator/commit/febe1d323bb80e4ffc0e36721fa1cd75a949b023))
* Disable SBOM upload-release-assets to avoid race condition ([d909808](https://github.com/openclaw-rocks/k8s-operator/commit/d909808d5aecde367748691604be1fbc152a6619))
* Downgrade to Go 1.23 for golangci-lint compatibility ([dd33c08](https://github.com/openclaw-rocks/k8s-operator/commit/dd33c08d1ffe962f358c14a6ae0a7b3105799d44))
* Increase golangci-lint timeout and update to v1.63.4 ([1d982ca](https://github.com/openclaw-rocks/k8s-operator/commit/1d982ca114f3316ad656bf873ad0fc6694f71cc5))
* Link OpenClaw to openclaw.ai, not openclaw.rocks ([679b043](https://github.com/openclaw-rocks/k8s-operator/commit/679b04340afc6835284f2e9e13e7aef02e69ed14))
* Link OpenClaw to openclaw.ai, not openclaw.rocks ([#22](https://github.com/openclaw-rocks/k8s-operator/issues/22)) ([9bfdbd8](https://github.com/openclaw-rocks/k8s-operator/commit/9bfdbd86b29ed43f043a16bdbd126f6960df32a2))
* Polish README copy and diagram alignment ([#19](https://github.com/openclaw-rocks/k8s-operator/issues/19)) ([ef6a8d1](https://github.com/openclaw-rocks/k8s-operator/commit/ef6a8d1f0973f36d3c790d34529f86e1c1d7343c))
* Pre-enable channel modules in config to prevent EBUSY on startup ([#13](https://github.com/openclaw-rocks/k8s-operator/issues/13)) ([21ee585](https://github.com/openclaw-rocks/k8s-operator/commit/21ee585fb4bef4f77afed0b0a8d730c0975af0e3)), closes [#11](https://github.com/openclaw-rocks/k8s-operator/issues/11)
* Prevent endless Deployment reconciliation loop ([#29](https://github.com/openclaw-rocks/k8s-operator/issues/29)) ([db942b9](https://github.com/openclaw-rocks/k8s-operator/commit/db942b945586e6c3b49db7f0d3ca242f5fac7b44)), closes [#28](https://github.com/openclaw-rocks/k8s-operator/issues/28)
* Remove chart-releaser, keep OCI-only Helm distribution ([#3](https://github.com/openclaw-rocks/k8s-operator/issues/3)) ([99e3cdb](https://github.com/openclaw-rocks/k8s-operator/commit/99e3cdba598ac46e5cd3cab2dc7dad00b456a0e8))
* Replace config subPath mount with init container to avoid EBUSY ([#10](https://github.com/openclaw-rocks/k8s-operator/issues/10)) ([38b60d3](https://github.com/openclaw-rocks/k8s-operator/commit/38b60d39b835fb1eca459f6c4d612d0304873c3e)), closes [#9](https://github.com/openclaw-rocks/k8s-operator/issues/9)
* Resolve variable shadowing of err in PVC reconciliation ([a9ea85b](https://github.com/openclaw-rocks/k8s-operator/commit/a9ea85b05770ba4123b4c0325e551ba93e90d103))
* Set HOME env var to match config mount path ([#5](https://github.com/openclaw-rocks/k8s-operator/issues/5)) ([175ff92](https://github.com/openclaw-rocks/k8s-operator/commit/175ff9219b5af8d80e37579627f8ebccce71cb1b)), closes [#4](https://github.com/openclaw-rocks/k8s-operator/issues/4)
* skip GitHub release creation in release-please ([#47](https://github.com/openclaw-rocks/k8s-operator/issues/47)) ([96a5be9](https://github.com/openclaw-rocks/k8s-operator/commit/96a5be93020ef61d6a37f804e1a741b3a229e86d))
* Specify kind cluster name for image loading ([7ac2116](https://github.com/openclaw-rocks/k8s-operator/commit/7ac21166b36e79fb0d93c23eb24de8c420dd8446))
* update banner alt text ([#38](https://github.com/openclaw-rocks/k8s-operator/issues/38)) ([d00dc23](https://github.com/openclaw-rocks/k8s-operator/commit/d00dc23516d79e6a95b6aeca9d5d29c1a777e0e6))
* Update Chart.yaml version/appVersion to match latest release v0.2.4 ([#26](https://github.com/openclaw-rocks/k8s-operator/issues/26)) ([c475ffc](https://github.com/openclaw-rocks/k8s-operator/commit/c475ffcd145b5109befb784a1a92672377be8936))
* Update copyright to 2026 OpenClaw.rocks ([bca1f0f](https://github.com/openclaw-rocks/k8s-operator/commit/bca1f0f626ebd8c7c1285882d20b87483380fba4))
* Update copyright to 2026 OpenClaw.rocks ([4d462f9](https://github.com/openclaw-rocks/k8s-operator/commit/4d462f97799865f48c13525074702f4b1163e54f))
* Update Go version to 1.24 for CI compatibility ([5c13d06](https://github.com/openclaw-rocks/k8s-operator/commit/5c13d068f3cba6d19609a3ea78b237d4b0b311ff))
* Use correct GitHub org name (OpenClaw-rocks) in all references ([#20](https://github.com/openclaw-rocks/k8s-operator/issues/20)) ([c157899](https://github.com/openclaw-rocks/k8s-operator/commit/c1578998cb7bfb4824eae64ce390fba4e508e6ca))
* Use direct append instead of loop for image pull secrets ([187477c](https://github.com/openclaw-rocks/k8s-operator/commit/187477c3e205ddf332bd9f2831936d141b1e4088))
* Use Go 1.24 with goinstall mode for golangci-lint ([3dd40c8](https://github.com/openclaw-rocks/k8s-operator/commit/3dd40c8fc02e964c9edd98407ae44a7d94634b2b))
* Use govet enable list for shadow analyzer ([3f04b1b](https://github.com/openclaw-rocks/k8s-operator/commit/3f04b1bd705f33df5c438e3ae9b22fd5e2cd7975))
* Use lowercase image names for OCI registry compatibility ([2d3c87e](https://github.com/openclaw-rocks/k8s-operator/commit/2d3c87ea562800a98b5be6c63b71bafe1fbae1e8))
* Use lowercase owner name for Helm OCI registry ([67bc33e](https://github.com/openclaw-rocks/k8s-operator/commit/67bc33e503a6f1329908dd437731b3bf96be2ea5))
* use PAT for release-please to trigger downstream workflows ([#45](https://github.com/openclaw-rocks/k8s-operator/issues/45)) ([21191ed](https://github.com/openclaw-rocks/k8s-operator/commit/21191ed3e29ff2d4aeb4c0aa269eca1fb7f6284a))

## [0.5.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.4.0...v0.5.0) (2026-02-13)


### ⚠ BREAKING CHANGES

* CRD API group changed from `openclaw.openclaw.io` to `openclaw.openclaw.rocks`. Existing CRDs must be deleted and re-created. This is acceptable at v1alpha1 stability level.

### Features

* Add nautical banner image with crab captain at Kubernetes helm ([84d1854](https://github.com/openclaw-rocks/k8s-operator/commit/84d1854cb7200587d6745c9ae3dc29c049fc3b8e))
* Add observability, testing, docs, and dev tooling ([5af27e1](https://github.com/openclaw-rocks/k8s-operator/commit/5af27e1884285d385356ee548f32a62dc8608a95))
* Add observability, testing, docs, and dev tooling for production readiness ([a8c063b](https://github.com/openclaw-rocks/k8s-operator/commit/a8c063b1e06143e06eacf84419ecbc31a9435684))
* Add support for custom sidecar containers ([#27](https://github.com/openclaw-rocks/k8s-operator/issues/27)) ([f0071f7](https://github.com/openclaw-rocks/k8s-operator/commit/f0071f72dd92b248223f53b7af9b2f3eb9a184be)), closes [#24](https://github.com/openclaw-rocks/k8s-operator/issues/24)
* Initial OpenClaw Kubernetes Operator implementation ([6d873ff](https://github.com/openclaw-rocks/k8s-operator/commit/6d873ff855273c9e51337181e657ced2b82af711))
* Inject CHROMIUM_URL env var into main container when sidecar is enabled ([8553cc7](https://github.com/openclaw-rocks/k8s-operator/commit/8553cc7ec0b41675bd815a0f8febd0622bbf068c))
* Inject CHROMIUM_URL env var when sidecar is enabled ([b80dabe](https://github.com/openclaw-rocks/k8s-operator/commit/b80dabe32dbf2ff0f755c92f4e83e2467050bd89))
* Replace manual release steps with GoReleaser ([#2](https://github.com/openclaw-rocks/k8s-operator/issues/2)) ([fedc497](https://github.com/openclaw-rocks/k8s-operator/commit/fedc49766d414e0a2d96bc9c080abcb82b2b2202))
* Support custom egress rules in NetworkPolicy ([#15](https://github.com/openclaw-rocks/k8s-operator/issues/15)) ([#16](https://github.com/openclaw-rocks/k8s-operator/issues/16)) ([c62dc09](https://github.com/openclaw-rocks/k8s-operator/commit/c62dc09cec6112cf4625175423d033310ce50dc3))
* update banner with real OpenClaw logo and Kubernetes logo ([#37](https://github.com/openclaw-rocks/k8s-operator/issues/37)) ([840adb9](https://github.com/openclaw-rocks/k8s-operator/commit/840adb9e285c14dffe1bfed17e9b9411b05f4538))


### Bug Fixes

* Add leader election RBAC and E2E test infrastructure ([4eb6019](https://github.com/openclaw-rocks/k8s-operator/commit/4eb601994f3b87bde07f4c28a51f1dd010f045d4))
* Apply same CreateOrUpdate pattern to ServiceMonitor reconciler ([#30](https://github.com/openclaw-rocks/k8s-operator/issues/30)) ([c1aaa36](https://github.com/openclaw-rocks/k8s-operator/commit/c1aaa3639b7cb82eb95696bbe06b98817d46a55e))
* Bump chromium /dev/shm from 256Mi to 1Gi ([925d7b8](https://github.com/openclaw-rocks/k8s-operator/commit/925d7b84c0630b73f581fca32d8da68ca582cd19))
* Bump chromium /dev/shm sizeLimit from 256Mi to 1Gi ([a4b3fbb](https://github.com/openclaw-rocks/k8s-operator/commit/a4b3fbb005fd40a743a54dcb8ce42c0db9ef7b27))
* change CRD API group domain from openclaw.io to openclaw.rocks ([#41](https://github.com/openclaw-rocks/k8s-operator/issues/41)) ([5bae852](https://github.com/openclaw-rocks/k8s-operator/commit/5bae852810d249358fad1453694503298851208c))
* Chromium sidecar crash (UID mismatch + read-only rootfs) and lint ([#14](https://github.com/openclaw-rocks/k8s-operator/issues/14)) ([febe1d3](https://github.com/openclaw-rocks/k8s-operator/commit/febe1d323bb80e4ffc0e36721fa1cd75a949b023))
* Disable SBOM upload-release-assets to avoid race condition ([d909808](https://github.com/openclaw-rocks/k8s-operator/commit/d909808d5aecde367748691604be1fbc152a6619))
* Downgrade to Go 1.23 for golangci-lint compatibility ([dd33c08](https://github.com/openclaw-rocks/k8s-operator/commit/dd33c08d1ffe962f358c14a6ae0a7b3105799d44))
* Increase golangci-lint timeout and update to v1.63.4 ([1d982ca](https://github.com/openclaw-rocks/k8s-operator/commit/1d982ca114f3316ad656bf873ad0fc6694f71cc5))
* Link OpenClaw to openclaw.ai, not openclaw.rocks ([679b043](https://github.com/openclaw-rocks/k8s-operator/commit/679b04340afc6835284f2e9e13e7aef02e69ed14))
* Link OpenClaw to openclaw.ai, not openclaw.rocks ([#22](https://github.com/openclaw-rocks/k8s-operator/issues/22)) ([9bfdbd8](https://github.com/openclaw-rocks/k8s-operator/commit/9bfdbd86b29ed43f043a16bdbd126f6960df32a2))
* Polish README copy and diagram alignment ([#19](https://github.com/openclaw-rocks/k8s-operator/issues/19)) ([ef6a8d1](https://github.com/openclaw-rocks/k8s-operator/commit/ef6a8d1f0973f36d3c790d34529f86e1c1d7343c))
* Pre-enable channel modules in config to prevent EBUSY on startup ([#13](https://github.com/openclaw-rocks/k8s-operator/issues/13)) ([21ee585](https://github.com/openclaw-rocks/k8s-operator/commit/21ee585fb4bef4f77afed0b0a8d730c0975af0e3)), closes [#11](https://github.com/openclaw-rocks/k8s-operator/issues/11)
* Prevent endless Deployment reconciliation loop ([#29](https://github.com/openclaw-rocks/k8s-operator/issues/29)) ([db942b9](https://github.com/openclaw-rocks/k8s-operator/commit/db942b945586e6c3b49db7f0d3ca242f5fac7b44)), closes [#28](https://github.com/openclaw-rocks/k8s-operator/issues/28)
* Remove chart-releaser, keep OCI-only Helm distribution ([#3](https://github.com/openclaw-rocks/k8s-operator/issues/3)) ([99e3cdb](https://github.com/openclaw-rocks/k8s-operator/commit/99e3cdba598ac46e5cd3cab2dc7dad00b456a0e8))
* Replace config subPath mount with init container to avoid EBUSY ([#10](https://github.com/openclaw-rocks/k8s-operator/issues/10)) ([38b60d3](https://github.com/openclaw-rocks/k8s-operator/commit/38b60d39b835fb1eca459f6c4d612d0304873c3e)), closes [#9](https://github.com/openclaw-rocks/k8s-operator/issues/9)
* Resolve variable shadowing of err in PVC reconciliation ([a9ea85b](https://github.com/openclaw-rocks/k8s-operator/commit/a9ea85b05770ba4123b4c0325e551ba93e90d103))
* Set HOME env var to match config mount path ([#5](https://github.com/openclaw-rocks/k8s-operator/issues/5)) ([175ff92](https://github.com/openclaw-rocks/k8s-operator/commit/175ff9219b5af8d80e37579627f8ebccce71cb1b)), closes [#4](https://github.com/openclaw-rocks/k8s-operator/issues/4)
* Specify kind cluster name for image loading ([7ac2116](https://github.com/openclaw-rocks/k8s-operator/commit/7ac21166b36e79fb0d93c23eb24de8c420dd8446))
* update banner alt text ([#38](https://github.com/openclaw-rocks/k8s-operator/issues/38)) ([d00dc23](https://github.com/openclaw-rocks/k8s-operator/commit/d00dc23516d79e6a95b6aeca9d5d29c1a777e0e6))
* Update Chart.yaml version/appVersion to match latest release v0.2.4 ([#26](https://github.com/openclaw-rocks/k8s-operator/issues/26)) ([c475ffc](https://github.com/openclaw-rocks/k8s-operator/commit/c475ffcd145b5109befb784a1a92672377be8936))
* Update copyright to 2026 OpenClaw.rocks ([bca1f0f](https://github.com/openclaw-rocks/k8s-operator/commit/bca1f0f626ebd8c7c1285882d20b87483380fba4))
* Update copyright to 2026 OpenClaw.rocks ([4d462f9](https://github.com/openclaw-rocks/k8s-operator/commit/4d462f97799865f48c13525074702f4b1163e54f))
* Update Go version to 1.24 for CI compatibility ([5c13d06](https://github.com/openclaw-rocks/k8s-operator/commit/5c13d068f3cba6d19609a3ea78b237d4b0b311ff))
* Use correct GitHub org name (OpenClaw-rocks) in all references ([#20](https://github.com/openclaw-rocks/k8s-operator/issues/20)) ([c157899](https://github.com/openclaw-rocks/k8s-operator/commit/c1578998cb7bfb4824eae64ce390fba4e508e6ca))
* Use direct append instead of loop for image pull secrets ([187477c](https://github.com/openclaw-rocks/k8s-operator/commit/187477c3e205ddf332bd9f2831936d141b1e4088))
* Use Go 1.24 with goinstall mode for golangci-lint ([3dd40c8](https://github.com/openclaw-rocks/k8s-operator/commit/3dd40c8fc02e964c9edd98407ae44a7d94634b2b))
* Use govet enable list for shadow analyzer ([3f04b1b](https://github.com/openclaw-rocks/k8s-operator/commit/3f04b1bd705f33df5c438e3ae9b22fd5e2cd7975))
* Use lowercase image names for OCI registry compatibility ([2d3c87e](https://github.com/openclaw-rocks/k8s-operator/commit/2d3c87ea562800a98b5be6c63b71bafe1fbae1e8))
* Use lowercase owner name for Helm OCI registry ([67bc33e](https://github.com/openclaw-rocks/k8s-operator/commit/67bc33e503a6f1329908dd437731b3bf96be2ea5))
* use PAT for release-please to trigger downstream workflows ([#45](https://github.com/openclaw-rocks/k8s-operator/issues/45)) ([21191ed](https://github.com/openclaw-rocks/k8s-operator/commit/21191ed3e29ff2d4aeb4c0aa269eca1fb7f6284a))

## [0.4.0](https://github.com/openclaw-rocks/k8s-operator/compare/v0.3.0...v0.4.0) (2026-02-13)


### ⚠ BREAKING CHANGES

* CRD API group changed from `openclaw.openclaw.io` to `openclaw.openclaw.rocks`. Existing CRDs must be deleted and re-created. This is acceptable at v1alpha1 stability level.

### Features

* Add nautical banner image with crab captain at Kubernetes helm ([84d1854](https://github.com/openclaw-rocks/k8s-operator/commit/84d1854cb7200587d6745c9ae3dc29c049fc3b8e))
* update banner with real OpenClaw logo and Kubernetes logo ([#37](https://github.com/openclaw-rocks/k8s-operator/issues/37)) ([840adb9](https://github.com/openclaw-rocks/k8s-operator/commit/840adb9e285c14dffe1bfed17e9b9411b05f4538))


### Bug Fixes

* Apply same CreateOrUpdate pattern to ServiceMonitor reconciler ([#30](https://github.com/openclaw-rocks/k8s-operator/issues/30)) ([c1aaa36](https://github.com/openclaw-rocks/k8s-operator/commit/c1aaa3639b7cb82eb95696bbe06b98817d46a55e))
* change CRD API group domain from openclaw.io to openclaw.rocks ([#41](https://github.com/openclaw-rocks/k8s-operator/issues/41)) ([5bae852](https://github.com/openclaw-rocks/k8s-operator/commit/5bae852810d249358fad1453694503298851208c))
* Prevent endless Deployment reconciliation loop ([#29](https://github.com/openclaw-rocks/k8s-operator/issues/29)) ([db942b9](https://github.com/openclaw-rocks/k8s-operator/commit/db942b945586e6c3b49db7f0d3ca242f5fac7b44)), closes [#28](https://github.com/openclaw-rocks/k8s-operator/issues/28)
* update banner alt text ([#38](https://github.com/openclaw-rocks/k8s-operator/issues/38)) ([d00dc23](https://github.com/openclaw-rocks/k8s-operator/commit/d00dc23516d79e6a95b6aeca9d5d29c1a777e0e6))

## [Unreleased]

### Added
- Custom Prometheus metrics (reconciliation duration, instance phases, resource failures)
- ServiceMonitor resource creation for Prometheus Operator integration
- Defaulting webhook for setting sensible defaults
- Comprehensive resource builder unit tests
- Webhook validation unit tests
- `.golangci.yaml` linter configuration
- `.dockerignore` for optimized Docker builds
- Architecture documentation
- API reference documentation
- Troubleshooting guide
- Deployment guides for EKS, GKE, AKS
- Grafana dashboard example
- PrometheusRule alert examples

## [0.1.0] - 2024-01-01

### Added
- Initial release of OpenClaw Kubernetes Operator
- OpenClawInstance CRD (v1alpha1)
- Controller with full reconciliation lifecycle
- Security-first design (non-root, dropped capabilities, seccomp, NetworkPolicy)
- Validating webhook (blocks root, warns on insecure config)
- Managed resources: Deployment, Service, ServiceAccount, Role, RoleBinding, NetworkPolicy, PDB, ConfigMap, PVC, Ingress
- Chromium sidecar support for browser automation
- Helm chart for installation
- CI/CD with GitHub Actions (lint, test, security scan, multi-arch build)
- Container image signing with Cosign
- SBOM generation
- E2E test infrastructure
