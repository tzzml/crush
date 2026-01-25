.PHONY: swagger swagger2 convert-openapi validate-openapi run build clean dev test

# 从代码生成 Swagger 2.0 文档,然后转换为 OpenAPI 3.0
swagger: swagger2 convert-openapi validate-openapi

# 从代码生成 Swagger 2.0 文档
swagger2:
	@echo "Generating Swagger 2.0 docs from code..."
	swag init -g cmd/server.go -o docs --parseDependency --parseInternal --outputTypes json,yaml
	@echo "✓ Swagger 2.0 docs generated"

# 将 Swagger 2.0 转换为 OpenAPI 3.0
convert-openapi:
	@echo "Converting to OpenAPI 3.0..."
	@python3 scripts/convert_to_openapi3.py docs/swagger.json docs/openapi3.json
	@echo "✓ OpenAPI 3.0 docs generated at docs/openapi3.json"

# 验证 OpenAPI 3.0 文档
validate-openapi:
	@echo "Validating OpenAPI 3.0 docs..."
	@scripts/validate_openapi3.sh

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
	rm -f docs/docs.go docs/swagger.json docs/swagger.yaml docs/openapi3.json

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
