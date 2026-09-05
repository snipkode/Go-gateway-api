# Local OIDC mock IdP for exercising the SSO flow without Azure.
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/mock-idp .

FROM alpine:3.20
COPY --from=builder /bin/mock-idp /bin/mock-idp
EXPOSE 9090
ENTRYPOINT ["/bin/mock-idp"]