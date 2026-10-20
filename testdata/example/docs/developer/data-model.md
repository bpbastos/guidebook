# Modelo de dados

```mermaid
erDiagram
  CUSTOMER ||--o{ ORDER : faz
  ORDER ||--|{ ITEM : contém
```
