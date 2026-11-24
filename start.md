cd C:\projects\mauzoPOS\backend
go run ./cmd/migrate up          # apply all migrations (incl. 00011 catalog)
go run ./cmd/migrate seed        # seed demo tenant, plans, business types, units
Start-Process powershell -ArgumentList '-NoExit','-Command','cd C:\projects\mauzoPOS\backend; go run ./cmd/api'
go build -o api.exe ./cmd/api
cd backend && ./api.exe

cd C:\projects\mauzoPOS\frontend
pnpm dev:web                     # serves the back-office UI on http://localhost:5173

Company: demo · Identity: owner@demo.local · Password: ChangeMe123!


netstat -ano | findstr :8080          # note the PID in the last column
taskkill /PID <pid> /F