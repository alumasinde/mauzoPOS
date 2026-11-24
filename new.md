
flowchart TD
    A[Cashier clicks Pay] --> B[Payment Drawer]
    B --> C{Payment Method}
    C -->|Cash| D[Cash Payment Service]
    C -->|M-Pesa| E[Daraja STK Push]
    C -->|Card| F[Card Payment Service]
    C -->|Bank| G[Bank Payment Service]
    E --> H[Payment Pending]
    H --> I[Daraja Callback]
    I --> J[Verify and Update Payment]
    D --> K[Payment Reconciliation]
    F --> K
    G --> K
    J --> K
    K --> L{Fully Paid?}
    L -->|Yes| M[Complete Sale]
    L -->|No| N[Outstanding Balance]
    M --> O[Generate Receipt and eTIMS Processing]
