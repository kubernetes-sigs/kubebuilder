TOOL=import-boss

all: test

test: build test_rules test_inverse

build:
	@go build -o /tmp/$(TOOL)

test_rules:
	  /tmp/$(TOOL) --logtostderr --v=4 -i $$(go list ./tests/rules/a)
	! /tmp/$(TOOL) --logtostderr --v=4 -i $$(go list ./tests/rules/b)
	  /tmp/$(TOOL) --logtostderr --v=4 -i $$(go list ./tests/rules/c)
	! /tmp/$(TOOL) --logtostderr --v=4 -i $$(go list ./tests/rules/nested)
	  /tmp/$(TOOL) --logtostderr --v=4 -i $$(go list ./tests/rules/nested/nested)
	! /tmp/$(TOOL) --logtostderr --v=4 -i $$(go list ./tests/rules/nested/nested/nested)
	! /tmp/$(TOOL) --logtostderr --v=4 -i $$(go list ./tests/rules/nested/nested/nested/inherit)

test_inverse:
	  /tmp/$(TOOL) --logtostderr --v=4 -i $$(go list ./tests/inverse/a ./tests/inverse/lib/... | grep -v quarantine | paste -sd',' -)
	! /tmp/$(TOOL) --logtostderr --v=4 -i $$(go list ./tests/inverse/b ./tests/inverse/lib/... | grep -v quarantine | paste -sd',' -)
	  /tmp/$(TOOL) --logtostderr --v=4 -i $$(go list ./tests/inverse/c ./tests/inverse/lib/... | grep -v quarantine | paste -sd',' -)
	! /tmp/$(TOOL) --logtostderr --v=4 -i $$(go list ./tests/inverse/d ./tests/inverse/lib/... | grep -v quarantine | paste -sd',' -)
	! /tmp/$(TOOL) --logtostderr --v=4 -i $$(go list ./tests/inverse/lib/... | paste -sd',' -)

	! /tmp/$(TOOL) --logtostderr --v=4 -i $$(go list ./tests/inverse/lib/... | paste -sd',' -)

.PHONY: build test test_rules test_inverse
