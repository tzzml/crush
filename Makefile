.PHONY: swagger run build clean dev test

# 生成 swagger 文档
swagger:
	@echo "Generating swagger docs..."
	swag init -g cmd/server.go -o docs --parseDependency --parseInternal
	@echo "✓ Swagger docs generated"

# 运行服务器
run:
	@echo "Starting server..."
	go run . serve

# 构建
build:
	@echo "Building..."
	go build -ldflags="-s -w" -o bin/zorkagent .

# 清理生成的文件
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f docs/docs.go docs/swagger.json docs/swagger.yaml

# 开发模式（生成文档 + 运行）
dev: swagger run

# 运行测试
test:
	@echo "Running tests..."
	go test ./...

# 安装依赖
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go install github.com/swaggo/swag/cmd/swag@latest
