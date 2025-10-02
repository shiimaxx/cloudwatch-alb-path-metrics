FROM golang:1.25.1 as builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bootstrap .

FROM public.ecr.aws/lambda/provided:al2

COPY --from=builder /src/bootstrap ${LAMBDA_RUNTIME_DIR}/

CMD ["bootstrap"]
