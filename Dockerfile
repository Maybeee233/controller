FROM golang:1.17 as builder

RUN go env -w GOPROXY=https://goproxy.cn,direct

WORKDIR /app

COPY . .
#CMD ["go env -w GOPROXY=https://goproxy.cn,direct"]



RUN CGO_ENABLED=0 go build -o ingress-manager main.go

FROM alpine:3.15.3

WORKDIR /app

COPY --from=builder /app/ingress-manager .

CMD ["./ingress-manager"]