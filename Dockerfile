# Imagem do guidebook serve (docs/spec.md, seção 14.4; ADR 0011).
# Bases fixadas por digest; troca de versão passa por ADR.

FROM golang:1.27.2-trixie@sha256:47667934874913a449b96faa7ef4777fa285d9ec29126ee80ed61c45b7cccf0e AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/guidebook ./cmd/guidebook

# Imagem final sem shell, com usuário não root e /tmp gravável (o único
# lugar onde o serve escreve).
FROM gcr.io/distroless/static-debian13:nonroot@sha256:516757106de35205efb70b91ba03f6f1298424a0678c603255cf2852a7000087
COPY --from=build /out/guidebook /guidebook
WORKDIR /project
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/guidebook", "serve", "-config", "/project/guidebook.yaml", "-addr", ":8080"]
