.PHONY: proto build test run clean

# Protocol Buffersのコンパイル
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/*.proto

# ビルド
build: proto
	go build -o bin/dtako_rows.exe cmd/server/main.go

# テスト
test:
	go test -v ./...

# カバレッジ付きテスト
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 実行
run: build
	./bin/dtako_rows.exe

# クリーンアップ
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	find proto -name "*.pb.go" -delete

# 依存関係の更新
deps:
	go mod download
	go mod tidy

# protoファイルのフォーマット
proto-fmt:
	@echo "Formatting proto files..."
	@for file in proto/*.proto; do \
		echo "Formatting $$file"; \
	done
