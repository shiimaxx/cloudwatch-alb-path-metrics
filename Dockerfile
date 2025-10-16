FROM golang:1.25 as builder

WORKDIR /src

COPY cmd/cloudwatch-alb-path-metrics/* ./

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bootstrap .

FROM public.ecr.aws/lambda/provided:al2

COPY --from=builder /src/bootstrap ${LAMBDA_RUNTIME_DIR}/

CMD ["bootstrap"]

