# Memória do Projeto - UnoGoBot

## Stack
- **Linguagem:** Go
- **Biblioteca Telegram:** telego (github.com/mymmrac/telego)
- **Estado:** Em memória (sem banco de dados)

## Arquitetura
- `main.go` — Entry point, inicializa bot e long polling
- `config.go` — Constantes de configuração (token, waiting_time, etc.)
- `card.go` — Definição de cartas, cores, valores, especiais, stickers
- `deck.go` — Baralho (shuffle, draw, dismiss, fill)
- `player.go` — Jogador (lista duplamente ligada em anel)
- `game.go` — Estado do jogo (regras, turnos, efeitos)
- `gamemanager.go` — Gerenciador de múltiplos jogos
- `results.go` — Construção de resultados inline
- `actions.go` — Ações do jogo (jogar carta, comprar, pular, etc.)
- `inline.go` — Handlers de inline query + chosen inline result
- `commands.go` — Handlers de comandos (/novo, /entrar, etc.)
- `errors.go` — Erros customizados

## Convenções
- Comandos em português (/novo, /entrar, /sair, etc.)
- Modo inline via stickers (usando file_ids do projeto jh0ker/mau_mau_bot)
- Jogador implementado como lista duplamente ligada em anel
- `sync.Mutex` por jogo para concorrência
- Context全局 compartilhado (`botCtx`)

## Decisões técnicas
- Usar telego ao invés de python-telegram-bot (migração Python → Go)
- Estado em memória (leve, sem dependências externas)
- Long polling (sem necessidade de webhook para dev)
- Stickers do projeto original (CARDS_CLASSIC_COLORBLIND)

## Problemas conhecidos
- Stickers podem não funcionar se o pacote de stickers original for alterado
- Modo de jogo "text" não implementado completamente (sempre usa stickers)
- Bot precisa de `/setinline` e `/setinlinefeedback` no BotFather
