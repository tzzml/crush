#!/usr/bin/env python3
"""
将 Swagger 2.0 规范转换为 OpenAPI 3.0 规范
"""

import json
import sys
from pathlib import Path


def convert_swagger_to_openapi3(swagger_data):
    """将 Swagger 2.0 转换为 OpenAPI 3.0"""

    openapi = {
        "openapi": "3.0.0",
        "info": swagger_data.get("info", {}),
        "paths": {},
        "components": {
            "schemas": {}
        }
    }

    # 转换 servers (从 host, basePath, schemes)
    if "host" in swagger_data or "basePath" in swagger_data or "schemes" in swagger_data:
        schemes = swagger_data.get("schemes", ["http"])
        host = swagger_data.get("host", "localhost:8080")
        base_path = swagger_data.get("basePath", "/")

        openapi["servers"] = []
        for scheme in schemes:
            openapi["servers"].append({
                "url": f"{scheme}://{host}{base_path}"
            })

    # 转换 paths
    if "paths" in swagger_data:
        for path, path_item in swagger_data["paths"].items():
            openapi["paths"][path] = {}

            for method, operation in path_item.items():
                if method.lower() not in ["get", "post", "put", "delete", "patch", "options", "head", "trace"]:
                    continue

                openapi_op = {}

                # 复制基本字段
                for field in ["tags", "summary", "description", "operationId", "deprecated"]:
                    if field in operation:
                        openapi_op[field] = operation[field]

                # 转换 parameters
                if "parameters" in operation:
                    openapi_op["parameters"] = []
                    for param in operation["parameters"]:
                        new_param = {
                            "name": param["name"],
                            "in": param["in"]
                        }

                        if "description" in param:
                            new_param["description"] = param["description"]
                        if "required" in param:
                            new_param["required"] = param["required"]
                        if "deprecated" in param:
                            new_param["deprecated"] = param["deprecated"]
                        if "allowEmptyValue" in param:
                            new_param["allowEmptyValue"] = param["allowEmptyValue"]

                        # 转换 schema
                        if "schema" in param:
                            new_param["schema"] = convert_schema(param["schema"])
                        elif "type" in param:
                            new_param["schema"] = {"type": param["type"]}

                        openapi_op["parameters"].append(new_param)

                # 转换 requestBody (从 in: body 的参数)
                body_params = [p for p in operation.get("parameters", []) if p.get("in") == "body"]
                if body_params:
                    # 转换 schema 中的引用
                    schema = convert_schema(body_params[0].get("schema", {}))
                    openapi_op["requestBody"] = {
                        "required": any(p.get("required", False) for p in body_params),
                        "content": {
                            "application/json": {
                                "schema": schema
                            }
                        }
                    }
                    # 移除 parameters 中的 body 参数
                    if "parameters" in openapi_op:
                        openapi_op["parameters"] = [
                            p for p in openapi_op["parameters"]
                            if p.get("in") != "body"
                        ]
                        if not openapi_op["parameters"]:
                            del openapi_op["parameters"]

                # 转换 responses
                if "responses" in operation:
                    openapi_op["responses"] = {}
                    for code, response in operation["responses"].items():
                        new_response = {}

                        if "description" in response:
                            new_response["description"] = response["description"]

                        # 转换 schema
                        if "schema" in response:
                            new_response["content"] = {
                                "application/json": {
                                    "schema": convert_schema(response["schema"])
                                }
                            }

                        # 处理 headers
                        if "headers" in response:
                            new_response["headers"] = response["headers"]

                        # 处理 examples
                        if "examples" in response:
                            new_response["content"]["application/json"]["examples"] = response["examples"]

                        openapi_op["responses"][str(code)] = new_response

                # 转换 security
                if "security" in operation:
                    openapi_op["security"] = operation["security"]

                # 转换 consumes/produces 到 requestBody/content
                if "consumes" in operation and "requestBody" in openapi_op:
                    # consumes 已经在 requestBody 中处理为 application/json
                    pass
                if "produces" in operation:
                    # produces 已经在 responses 中处理为 application/json
                    pass

                openapi["paths"][path][method] = openapi_op

    # 转换 definitions 到 components/schemas
    if "definitions" in swagger_data:
        for name, schema in swagger_data["definitions"].items():
            openapi["components"]["schemas"][name] = convert_schema(schema)

    # 转换 securityDefinitions
    if "securityDefinitions" in swagger_data:
        openapi["components"]["securitySchemes"] = swagger_data["securityDefinitions"]

    # 转换 security
    if "security" in swagger_data:
        openapi["security"] = swagger_data["security"]

    # 转换 tags
    if "tags" in swagger_data:
        openapi["tags"] = swagger_data["tags"]

    # 转换 externalDocs
    if "externalDocs" in swagger_data:
        openapi["externalDocs"] = swagger_data["externalDocs"]

    return openapi


def convert_schema(schema):
    """递归转换 schema 中的 $ref 引用"""
    if not isinstance(schema, dict):
        return schema

    new_schema = {}

    for key, value in schema.items():
        if key == "$ref":
            # 转换引用路径
            if value.startswith("#/definitions/"):
                new_schema["$ref"] = value.replace("#/definitions/", "#/components/schemas/")
            else:
                new_schema["$ref"] = value
        elif isinstance(value, dict):
            new_schema[key] = convert_schema(value)
        elif isinstance(value, list):
            new_schema[key] = [
                convert_schema(item) if isinstance(item, dict) else item
                for item in value
            ]
        else:
            new_schema[key] = value

    return new_schema


def main():
    if len(sys.argv) < 2:
        print("Usage: python convert_to_openapi3.py <input.json> [output.json]")
        sys.exit(1)

    input_file = Path(sys.argv[1])
    output_file = Path(sys.argv[2]) if len(sys.argv) > 2 else input_file.parent / f"{input_file.stem}-openapi3.json"

    # 读取 Swagger 2.0 文件
    with open(input_file, 'r', encoding='utf-8') as f:
        swagger_data = json.load(f)

    # 验证是 Swagger 2.0
    if swagger_data.get("swagger") != "2.0":
        print(f"Warning: {input_file} may not be a Swagger 2.0 file")

    # 转换为 OpenAPI 3.0
    openapi_data = convert_swagger_to_openapi3(swagger_data)

    # 写入输出文件
    with open(output_file, 'w', encoding='utf-8') as f:
        json.dump(openapi_data, f, ensure_ascii=False, indent=2)

    print(f"✓ Converted {input_file} to {output_file}")
    print(f"  Format: OpenAPI {openapi_data['openapi']}")
    print(f"  Endpoints: {len(openapi_data['paths'])}")
    print(f"  Schemas: {len(openapi_data['components']['schemas'])}")


if __name__ == "__main__":
    main()
