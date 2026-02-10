.PHONY: build dev frontend clean

# Build the complete application (frontend + backend)
build: frontend
	go build -o lark-robot.exe .

# Run backend in development mode
dev:
	go run . -config config.yaml

# Build Vue frontend and copy to static/dist
frontend:
	cd web && npm install && npm run build
	@if exist static\dist rmdir /s /q static\dist
	xcopy /E /I /Y web\dist static\dist

# Clean build artifacts
clean:
	@if exist lark-robot.exe del lark-robot.exe
	@if exist static\dist rmdir /s /q static\dist
	@if exist web\dist rmdir /s /q web\dist
	@if exist web\node_modules rmdir /s /q web\node_modules

# Run frontend dev server (with proxy to backend)
frontend-dev:
	cd web && npm run dev

# Download Go dependencies
deps:
	go mod tidy
