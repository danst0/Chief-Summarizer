BINARY := chief-summarizer
CMD_DIR := ./cmd/chief-summarizer
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: build install clean

build:
	@echo "Building $(BINARY)..."
	@go build -o $(BINARY) $(CMD_DIR)

install: build
	@echo "Installing $(BINARY) to $(INSTALL_DIR)"
	@install -d $(INSTALL_DIR)
	@install -m 755 $(BINARY) $(INSTALL_DIR)/$(BINARY)

clean:
	@rm -f $(BINARY)
