.PHONY: hooks build test preview

# Ativa os hooks versionados em .githooks/.
hooks:
	git config core.hooksPath .githooks

# Compila o binário em bin/.
build:
	go build -o bin/guidebook ./cmd/guidebook

# Roda todos os testes.
test:
	go test ./...

# Gera em out/ os guias do próprio guidebook e os do projeto de exemplo,
# para abrir no navegador com dois cliques.
preview: build
	bin/guidebook build -out out/guidebook
	bin/guidebook build -config testdata/example/guidebook.yaml -out out/example
	@echo
	@echo "Abra no navegador:"
	@for f in out/guidebook/user-guide.html out/guidebook/developer-guide.html \
		out/example/user-guide.html out/example/developer-guide.html; do \
		echo "  file://$$(pwd)/$$f"; \
	done
