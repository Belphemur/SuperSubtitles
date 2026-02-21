# [2.0.0](https://github.com/Belphemur/SuperSubtitles/compare/v1.8.0...v2.0.0) (2026-02-21)


### Features

* **services:** change GetRecentSubtitles to stream ShowSubtitlesCollection and remove ShowSubtitleItem ([45b8a84](https://github.com/Belphemur/SuperSubtitles/commit/45b8a84bcfac06ca6b4394c4877d10cdb610773e))
* **services:** refactor GetShowSubtitles and StreamRecentSubtitles to stream complete ShowSubtitles collections ([b08fbc3](https://github.com/Belphemur/SuperSubtitles/commit/b08fbc3862c8a12743b38e1eece21309e45730ca))


### BREAKING CHANGES

* **services:** GetRecentSubtitles now returns stream ShowSubtitlesCollection instead of stream ShowSubtitleItem. The ShowSubtitleItem proto message and models.ShowInfo/ShowSubtitleItem Go types have been removed.

Co-authored-by: Belphemur <197810+Belphemur@users.noreply.github.com>
* **services:** GetShowSubtitles now returns stream ShowSubtitlesCollection instead of stream ShowSubtitleItem. Client StreamShowSubtitles and StreamRecentSubtitles now return ShowSubtitles (accumulated per show) instead of individual ShowSubtitleItem messages.

Co-authored-by: Belphemur <197810+Belphemur@users.noreply.github.com>

# [1.8.0](https://github.com/Belphemur/SuperSubtitles/compare/v1.7.0...v1.8.0) (2026-02-20)


### Bug Fixes

* **client:** log warning when show parsing fails in streamShowsFromBody ([ca7d2ab](https://github.com/Belphemur/SuperSubtitles/commit/ca7d2abff8263860b584c54fe03a3ec37e8f6d71))


### Features

* **client:** parallel pagination for show list endpoints ([719ab3c](https://github.com/Belphemur/SuperSubtitles/commit/719ab3c3a1e5478a7e90cb524a060debf68063cf))
* **parser:** add ExtractLastPage for show list pagination ([364a862](https://github.com/Belphemur/SuperSubtitles/commit/364a862574cbe3e0beecfbc2cecc8862d94b87a3))

# [1.7.0](https://github.com/Belphemur/SuperSubtitles/compare/v1.6.5...v1.7.0) (2026-02-19)


### Bug Fixes

* **client:** handle string values in update check JSON response ([43ebb4d](https://github.com/Belphemur/SuperSubtitles/commit/43ebb4d52dfa834f9e22cd8114b28c07347da8d6))


### Features

* change CheckForUpdates content ID parameter from string to int64 ([edf426c](https://github.com/Belphemur/SuperSubtitles/commit/edf426cb869101182d569ae2545551f3a0dfa28f))
* change CheckForUpdatesRequest.content_id from string to int64 ([f11af7d](https://github.com/Belphemur/SuperSubtitles/commit/f11af7d82b72a5e3b03492c9cbcdd80c1460cdd0))

## [1.6.5](https://github.com/Belphemur/SuperSubtitles/compare/v1.6.4...v1.6.5) (2026-02-19)


### Bug Fixes

* **parser:** deduplicate release groups case-insensitively ([7cf66c0](https://github.com/Belphemur/SuperSubtitles/commit/7cf66c0e45ecc19217cfdb7a4e8325dccec920d7))

## [1.6.4](https://github.com/Belphemur/SuperSubtitles/compare/v1.6.3...v1.6.4) (2026-02-19)


### Bug Fixes

* **parser:** trim trailing punctuation in episode titles without SxEE pattern ([422e17b](https://github.com/Belphemur/SuperSubtitles/commit/422e17ba3aac631caab40954c9126dfca27a344a))

## [1.6.3](https://github.com/Belphemur/SuperSubtitles/compare/v1.6.2...v1.6.3) (2026-02-19)


### Bug Fixes

* **parser:** extract only episode title for subtitle names ([fdce830](https://github.com/Belphemur/SuperSubtitles/commit/fdce830244f30867701239f9abdc66004428617a))
* **parser:** improve episode title extraction logic ([f22cdb1](https://github.com/Belphemur/SuperSubtitles/commit/f22cdb1cb665b8725959dc3c4eda172fe01495aa))

## [1.6.2](https://github.com/Belphemur/SuperSubtitles/compare/v1.6.1...v1.6.2) (2026-02-18)


### Bug Fixes

* use proper environment variables. ([d9277c7](https://github.com/Belphemur/SuperSubtitles/commit/d9277c738a7d10ca73288c83550fb595102a874f))

## [1.6.1](https://github.com/Belphemur/SuperSubtitles/compare/v1.6.0...v1.6.1) (2026-02-18)


### Bug Fixes

* **docker:** set proper server address and port ([3bbdc1d](https://github.com/Belphemur/SuperSubtitles/commit/3bbdc1d056c179afe252f38e7a43a9d10077b7d3))

# [1.6.0](https://github.com/Belphemur/SuperSubtitles/compare/v1.5.1...v1.6.0) (2026-02-17)


### Features

* **grpc:** add standard gRPC health check support ([#34](https://github.com/Belphemur/SuperSubtitles/issues/34)) ([97ce131](https://github.com/Belphemur/SuperSubtitles/commit/97ce131522ec1f11d50b287c4ba80c3f6daa2318)), closes [#36](https://github.com/Belphemur/SuperSubtitles/issues/36)

## [1.5.1](https://github.com/Belphemur/SuperSubtitles/compare/v1.5.0...v1.5.1) (2026-02-17)


### Bug Fixes

* **client:** client methods to use streaming results ([c2045dc](https://github.com/Belphemur/SuperSubtitles/commit/c2045dcc6597921131a4a90313db39e023333f8e))
* **subtitle:** discard invalid subtitles with ID <=  0 ([708d0de](https://github.com/Belphemur/SuperSubtitles/commit/708d0de19d3dd5269c849cb7de04f95977638d0a))

# [1.5.0](https://github.com/Belphemur/SuperSubtitles/compare/v1.4.4...v1.5.0) (2026-02-14)


### Features

* **client:** add channel-based streaming methods for reduced memory usage ([48d5313](https://github.com/Belphemur/SuperSubtitles/commit/48d531329c65b52d0b4356369ee7606ad84b2c9e))
* **grpc:** convert RPCs to server-side streaming with proto and server changes ([3b4f68c](https://github.com/Belphemur/SuperSubtitles/commit/3b4f68c29fd664a11eae66a7b3e31be2a99be654))
* **grpc:** stream ShowSubtitleItem for GetRecentSubtitles with show info caching ([0952546](https://github.com/Belphemur/SuperSubtitles/commit/0952546d61273d614b46023bee8f20b025031d97))
* **grpc:** stream Subtitle directly for GetRecentSubtitles instead of ShowSubtitleItem ([678c281](https://github.com/Belphemur/SuperSubtitles/commit/678c281823c9c7f7b66496b79176b078f329c5ba))

## [1.4.4](https://github.com/Belphemur/SuperSubtitles/compare/v1.4.3...v1.4.4) (2026-02-13)


### Bug Fixes

* **deps:** update module google.golang.org/grpc to v1.79.1 ([#28](https://github.com/Belphemur/SuperSubtitles/issues/28)) ([4e18e0f](https://github.com/Belphemur/SuperSubtitles/commit/4e18e0f46e07cf43b6907c7b26a8ecc642fec594))

## [1.4.3](https://github.com/Belphemur/SuperSubtitles/compare/v1.4.2...v1.4.3) (2026-02-12)


### Bug Fixes

* **deps:** update module google.golang.org/grpc to v1.79.0 ([#27](https://github.com/Belphemur/SuperSubtitles/issues/27)) ([d1dd3b0](https://github.com/Belphemur/SuperSubtitles/commit/d1dd3b0f7a156cc1a84306ce8a99661aa6e6a8e8))

## [1.4.2](https://github.com/Belphemur/SuperSubtitles/compare/v1.4.1...v1.4.2) (2026-02-10)


### Bug Fixes

* **proto:** make episode optional in DownloadSubtitleRequest  ([dca2cf2](https://github.com/Belphemur/SuperSubtitles/commit/dca2cf2adead69bf9994c715cd0e809bc6436cec)), closes [#25](https://github.com/Belphemur/SuperSubtitles/issues/25)

## [1.4.1](https://github.com/Belphemur/SuperSubtitles/compare/v1.4.0...v1.4.1) (2026-02-10)


### Bug Fixes

* **parser:** normalize download URLs to prevent double-encoding in JSON ([94defa7](https://github.com/Belphemur/SuperSubtitles/commit/94defa7baad0fdc9f8b099c21680b82ba172d245))

# [1.4.0](https://github.com/Belphemur/SuperSubtitles/compare/v1.3.0...v1.4.0) (2026-02-09)


### Bug Fixes

* **grpc:** add nil check for show entries in GetShowSubtitles ([f696527](https://github.com/Belphemur/SuperSubtitles/commit/f6965276fc019e354a3b38718c320be69d6407cd))
* **grpc:** add safe int conversion to prevent overflow in proto mappings ([499d087](https://github.com/Belphemur/SuperSubtitles/commit/499d087c188725ff2dd17976be0c15dc2b781276))
* **parser:** simplify show name extraction using DOM sibling traversal ([c4aaee4](https://github.com/Belphemur/SuperSubtitles/commit/c4aaee4f3792ae69d78f703bfa827d1d034bdccf))
* use go generate for pro generation ([4865d99](https://github.com/Belphemur/SuperSubtitles/commit/4865d994a32ff9a18caa30849d325710d2ce4b45))


### Features

* **grpc:** add complete gRPC API with proto definitions and server implementation ([a23fa78](https://github.com/Belphemur/SuperSubtitles/commit/a23fa78375cccb418b9548e7269e967d07eb5abf))

# [1.3.0](https://github.com/Belphemur/SuperSubtitles/compare/v1.2.1...v1.3.0) (2026-02-09)


### Bug Fixes

* **client:** correct test struct field names to match actual API ([cbc3030](https://github.com/Belphemur/SuperSubtitles/commit/cbc3030b19fba4085ca8fa16d98643d178c7bc64))


### Features

* **client:** add GetRecentSubtitles API with ID filtering ([1097c5f](https://github.com/Belphemur/SuperSubtitles/commit/1097c5f57d5acf875fc7a8452ee8b182b202b99e))
* **parser:** remove parenthetical content from subtitle names ([65acdef](https://github.com/Belphemur/SuperSubtitles/commit/65acdefa8db0bd56adfc7de3c41caae2996fb831))


### Performance Improvements

* improve performance of recent subtitles ([fb4bb57](https://github.com/Belphemur/SuperSubtitles/commit/fb4bb57e4b5ee17ef6b4f73834cc1303066f7e09))

## [1.2.1](https://github.com/Belphemur/SuperSubtitles/compare/v1.2.0...v1.2.1) (2026-02-09)


### Performance Improvements

* **client:** optimize third-party ID fetching to use first page only ([7dce918](https://github.com/Belphemur/SuperSubtitles/commit/7dce9183be6a15a5e2e660e90a38df0bb1333aae))

# [1.2.0](https://github.com/Belphemur/SuperSubtitles/compare/v1.1.2...v1.2.0) (2026-02-09)


### Features

* **parser:** convert language names to ISO 639-1 codes ([d8603bf](https://github.com/Belphemur/SuperSubtitles/commit/d8603bfc684294d16d053d08e674aef293cab340))

## [1.1.2](https://github.com/Belphemur/SuperSubtitles/compare/v1.1.1...v1.1.2) (2026-02-09)


### Bug Fixes

* **subtitle:** fix getting proper subtitle info ([f8ac0a0](https://github.com/Belphemur/SuperSubtitles/commit/f8ac0a094f3a9a0a0637bbf958086ac2883a1b6c))

## [1.1.1](https://github.com/Belphemur/SuperSubtitles/compare/v1.1.0...v1.1.1) (2026-02-09)


### Bug Fixes

* **deps:** update module github.com/klauspost/compress to v1.18.4 ([#14](https://github.com/Belphemur/SuperSubtitles/issues/14)) ([080ca22](https://github.com/Belphemur/SuperSubtitles/commit/080ca229e573ac89c0dff2a23cc0a09807f84fa3))

# [1.1.0](https://github.com/Belphemur/SuperSubtitles/compare/v1.0.2...v1.1.0) (2026-02-09)


### Features

* **client:** fetch subtitles via HTML pagination ([#11](https://github.com/Belphemur/SuperSubtitles/issues/11)) ([d7b8c78](https://github.com/Belphemur/SuperSubtitles/commit/d7b8c78d27573f405d5326a2f5ee3096a242e905))
* configurable user agent ([aa8c800](https://github.com/Belphemur/SuperSubtitles/commit/aa8c800d7ddb1a200bc1b2d398ed33c038880f15))

## [1.0.2](https://github.com/Belphemur/SuperSubtitles/compare/v1.0.1...v1.0.2) (2026-02-09)


### Bug Fixes

* building docker image ([3f88452](https://github.com/Belphemur/SuperSubtitles/commit/3f884527b9853d2a227412d12f5c9100930c9a77))

## [1.0.1](https://github.com/Belphemur/SuperSubtitles/compare/v1.0.0...v1.0.1) (2026-02-09)


### Bug Fixes

* proper namspace ([9af575c](https://github.com/Belphemur/SuperSubtitles/commit/9af575cec5e5fac1ea7d17afc24f3a303c98171a))

# 1.0.0 (2026-02-09)


### Bug Fixes

* **ci:** add explicit permissions block to CI workflow ([9048de3](https://github.com/Belphemur/SuperSubtitles/commit/9048de3c2d0c604c5880a698a0ae6e40e7ec20c0))
* **client:** add other page with shows ([b19c9dd](https://github.com/Belphemur/SuperSubtitles/commit/b19c9ddb1689ddd22a7befe337f7bafde03d37f1))
* **client:** handle edge cases in compression transport ([430a3b1](https://github.com/Belphemur/SuperSubtitles/commit/430a3b1b6359ec0705861ac153d4f243827d79b5))
* **lint:** resolve golangci-lint errcheck and staticcheck issues ([0d2ac63](https://github.com/Belphemur/SuperSubtitles/commit/0d2ac63874ad2d857f902e51e85e77773fbde04a))
* not finding all shows ([d59509c](https://github.com/Belphemur/SuperSubtitles/commit/d59509c15ff0ca5075d1b2623f8c4acd25d476d0))
* **services:** add download size limit and subtitle file prioritization ([ca3d850](https://github.com/Belphemur/SuperSubtitles/commit/ca3d85061ae0bf5c99f70505279313c4265a44b0))
* **services:** address PR review feedback ([01e0873](https://github.com/Belphemur/SuperSubtitles/commit/01e0873d1eb9b469e73480cae015f368190a080d))
* **services:** improve content-type pattern matching specificity ([c43044a](https://github.com/Belphemur/SuperSubtitles/commit/c43044ae8e0830a6a0a4a04a9c2fd484ab05e1be))
* **services:** improve episode extraction and centralize User-Agent ([bc34947](https://github.com/Belphemur/SuperSubtitles/commit/bc3494743cc9fb5458d94974ed0056a6d115fc65))
* **services:** use ZIP magic number detection and proper MIME parsing ([502eb7c](https://github.com/Belphemur/SuperSubtitles/commit/502eb7ce5cff3307c39c27f40c388bef6adaaa33))
* **tests:** properly create ZIP in benchmark test ([633f0f6](https://github.com/Belphemur/SuperSubtitles/commit/633f0f6722a22078d197f683cfad7d4a99982f31))


### Features

* add parsing of subtitle ([0323b11](https://github.com/Belphemur/SuperSubtitles/commit/0323b11e62bfbf7d77aeb062b1d41724b74140fc))
* **ci:** add CI/CD pipeline with linting, testing, and semantic-release ([14e6922](https://github.com/Belphemur/SuperSubtitles/commit/14e69227674751585bee714d28eb290ed629199e))
* **client:** add GetShowSubtitles method with parallel processing ([c2cc935](https://github.com/Belphemur/SuperSubtitles/commit/c2cc93572322aadc957aab490d98566e70caf8da))
* **client:** add HTTP compression support for gzip, brotli, and zstd ([f25af39](https://github.com/Belphemur/SuperSubtitles/commit/f25af39b8b11dad619bfdc83ac1ca4edaa69cf71))
* **client:** fetch show list from both waiting and under-translation endpoints in parallel ([e4da04b](https://github.com/Belphemur/SuperSubtitles/commit/e4da04b8944a5d0e0093bef85b4969209c91df72))
* **docker:** add multi-platform Docker image builds to GoReleaser ([1bb18ee](https://github.com/Belphemur/SuperSubtitles/commit/1bb18ee3317139f97624324ceee97920bd516836))
* extract shows from html ([335354c](https://github.com/Belphemur/SuperSubtitles/commit/335354c2b65b1f8cb10b0d1ccf4d265ac698912c))
* fetch all subtitle of a show ([dacb07f](https://github.com/Belphemur/SuperSubtitles/commit/dacb07fe171cf9f665f57d5e3bbe69e22841750c))
* get subtitles data for show ([d700aa8](https://github.com/Belphemur/SuperSubtitles/commit/d700aa8cc376a8a9f231acf1281dbef2650604be))
* **models:** add ShowSubtitles model with comprehensive third-party IDs support ([11c150a](https://github.com/Belphemur/SuperSubtitles/commit/11c150ab23e7d76574e328e0aa7fb193189d4c8b))
* parse subtitle JSON from xmbc for a show ([515b1b7](https://github.com/Belphemur/SuperSubtitles/commit/515b1b7ab085dc588acb4d7163ee68836bdcb0af))
* **parser:** add third-party ID parser for extracting IMDB, TVDB, TVMaze, and Trakt IDs from HTML ([b0a6739](https://github.com/Belphemur/SuperSubtitles/commit/b0a6739dc1aef2d0176c3f5566e94877bc222b70))
* **services:** add subtitle download with season pack episode extraction ([73431ed](https://github.com/Belphemur/SuperSubtitles/commit/73431ed83ce1ec02e3c2851ce6dbd59903e6f604))
* **services:** add ZIP bomb detection for subtitle downloads ([3c51d86](https://github.com/Belphemur/SuperSubtitles/commit/3c51d868e34bccf6bb5f40f5c0506e6b3c966d8a))
* **updates:** add way to check if there are update since a specific episode id ([0db2d0c](https://github.com/Belphemur/SuperSubtitles/commit/0db2d0c5b015a48dc76f96e9482c1f0760b7f517))
