# Plano: substituir-start-por-iniciar

## Pedido do usuário
Trocar o comando `/start@usernamebot` para `/iniciar@usernamebot`, adicionar mensagem "falta x jogadores para iniciar" quando o número de jogadores for menor que o mínimo, incluir `/iniciar` na mensagem de entrada no jogo e na mensagem de alerta de mínimo de jogadores.

## Objetivo
Alterar o comando de início de jogo de `/start` para `/iniciar`, melhorar as mensagens de feedback para o usuário indicando quantos jogadores faltam para começar.

## Contexto atual
Atualmente o bot usa `/start` para iniciar partidas em grupo. As mensagens de novo jogo, entrada no jogo, ajuda, e alerta de mínimo jogadores referenciam `/start`. A mensagem de mínimo jogadores apenas diz "Pelo menos 2 jogadores devem entrar no jogo" sem informar quantos faltam.

## Arquivos analisados
- commands.go (handler de comandos, mensagens de ajuda)
- results.go (mensagens inline)
- main.go (registro de comandos do bot)
- config.go (configuração de mínimo de jogadores)
- game.go (lógica do jogo)
- gamemanager.go (gerenciamento de jogos)
- player.go (jogadores)
- inline.go (inline queries)

## Arquivos que serão modificados
- commands.go
- results.go
- main.go

## Estratégia de implementação
1. Adicionar `"/iniciar"` como case no switch de comandos (commands.go), mantendo `/start` como alias para não quebrar compatibilidade
2. Alterar todas as mensagens de texto que referenciam `/start` para `/iniciar`
3. Na verificação de mínimo de jogadores, calcular e exibir quantos faltam
4. Na mensagem de entrada no jogo, adicionar referência ao `/iniciar`
5. Atualizar o registro de comandos do bot em main.go

## Passos detalhados

1. **commands.go**: Adicionar `case "/iniciar": cmdStartGame(...)` ao switch
2. **commands.go**: Na `cmdNewGame` (linha 105), trocar `/start` por `/iniciar`
3. **commands.go**: Na `cmdJoinGame` (linha 117), melhorar mensagem: "Você entrou no jogo!" → incluir `/iniciar` e info de quantos jogadores faltam
4. **commands.go**: Na `cmdStartGame` (linhas 167-169), melhorar mensagem de mínimo jogadores: informar quantos faltam e usar `/iniciar`
5. **commands.go**: Na `cmdStartGame` (linhas 131-153), trocar referências a `/start` por `/iniciar` na mensagem privada
6. **commands.go**: Na `cmdKillGame` (linha 258), trocar `/start` por `/iniciar`
7. **commands.go**: Na `cmdHelp` (linhas 450-476), trocar todas as referências de `/start` para `/iniciar`
8. **results.go**: Na `addNotStarted` (linha 180), trocar `/start` por `/iniciar`
9. **main.go**: No registro de comandos (linha 56), trocar `/start` por `/iniciar`

## Riscos
- Usuários acostumados com `/start` podem estranhar a mudança (mantido como alias)
- Mensagens hardcoded espalhadas pelo código (todas mapeadas)
- Nenhum risco técnico significativo

## Impactos esperados
- Comando principal muda de `/start` para `/iniciar`
- Mensagens mais informativas sobre quantos jogadores faltam
- Melhor experiência para novos usuários

## Compatibilidade
- Linux ✓
- macOS ✓
- Windows ✓
- Docker ✓
- CI/CD ✓

## Como testar

### Build
```bash
go build -o unobot .
```

### Testes
```bash
go vet ./...
```

### Execução
```bash
./unobot
```

## Rollback
Reverter alterações nos arquivos commands.go, results.go e main.go usando `git checkout`.

## Observações
- `/start` continua funcionando como alias para não quebrar compatibilidade
- O bot username (`@usernamebot`) é dinâmico e resolvido em runtime, então os comandos nas mensagens são sempre sem o `@`
