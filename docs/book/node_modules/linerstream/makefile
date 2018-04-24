MOCHA?=node_modules/.bin/mocha
REPORTER?=spec
GROWL?=--growl
FLAGS=$(GROWL) --reporter $(REPORTER) --colors --bail

test:
	@NODE_ENV="test" \
	$(MOCHA) $(shell find test -name "*-test.js") $(FLAGS)

one:
	@NODE_ENV="test" \
	$(MOCHA) $(NAME) $(FLAGS)

.PHONY: test
