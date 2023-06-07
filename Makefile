test: mocks
	go test \
		-count=1 \
		-cover \
		-race \
		-timeout 60s

mocks:
	@rm -fr mocks
	mockery \
		--dir=. \
		--name=State \
		--structname=StmState \
		--outpkg mocks \
		--output ./mocks \
		--filename stm_state_mock.go 

.PHONY: test mocks