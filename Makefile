.PHONY: profile
profile:
	@mkdir -p pprof/
	go test -cpuprofile pprof/cpu.prof -memprofile pprof/mem.prof -bench .
	go tool pprof -pdf pprof/cpu.prof > pprof/cpu.pdf
	xdg-open pprof/cpu.pdf
	go tool pprof -weblist=.* pprof/cpu.prof

.PHONY: benchcmp
benchcmp:
	# ensure no govenor weirdness
	# sudo cpufreq-set -g performance
	go test -test.benchmem=true -run=NONE -bench=. ./... > bench_current.test
	git stash save "stashing for benchcmp"
	@go test -test.benchmem=true -run=NONE -bench=. ./... > bench_head.test
	git stash pop
	benchcmp bench_head.test bench_current.test
