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
