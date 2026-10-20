# Visão geral

O Pedido Fácil é uma aplicação web com um banco de dados.

```mermaid
C4Context
  Person(attendant, "Atendente")
  System(app, "Pedido Fácil")
  Rel(attendant, app, "Registra pedidos")
```
