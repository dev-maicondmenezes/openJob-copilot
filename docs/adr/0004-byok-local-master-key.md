# ADR-0004 — Criptografia BYOK: chave local em arquivo `0600` (sem senha mestra)

- **Status:** aceito
- **Data:** 2026-11-13
- **Contexto:** as chaves de API do usuário (BYOK, spec §3.2) precisam ficar cifradas em repouso no SQLite (spec §4). As opções eram (a) senha mestra digitada pelo usuário + derivação PBKDF2 a cada boot, ou (b) chave aleatória local gerada automaticamente e guardada em arquivo com permissão restrita.
- **Decisão:** adotar **AES-256-GCM** com uma **master key de 32 bytes aleatórios** gerada na primeira execução e persistida em `data/.master.key` (`0600`), ao lado do banco. O nonce é aleatório por cifração e o resultado é armazenado como `base64(nonce‖ciphertext)` na tabela `settings`. Não há senha mestra no MVP.
- **Alternativas descartadas:**
  - Senha mestra + PBKDF2: exige UX extra (digitar senha a cada boot) e tratamento de "esqueci a senha"; para uso pessoal single-user o arquivo local é equivalente em segurança prática e mais simples.
  - Chave derivada do hostname ou valor fixo no código: qualquer pessoa com acesso ao repo/DB decifraria — não é criptografia.
  - Keyring do SO: adiciona dependência de plataforma e quebra o modelo "um volume Docker = tudo do app".
- **Riscos conhecidos / decisão sobre perda da chave:** se `.master.key` for perdida ou regenerada, os valores cifrados ficam **irrecuperáveis** — isso é um erro **explícito** (`ErrKeyMismatch`), nunca uma regeneração silenciosa: `LoadOrCreateMasterKey` falha alto se o arquivo existir com tamanho diferente de 32 bytes. A recuperação é sempre: reconfigurar a API key pela UI (únicos dados afetados). Documentado no README: `.master.key` é o segredo do cofre — backup junto com o DB, nunca comitar.
- **Consequência:** inicialização zero-fricção (sem senha), criptografia autenticada (tamper-proof), e um único segredo de arquivo para proteger.
