## 📋 APIs e Fontes de Descoberta de Vagas Remotas — Lista Expandida

### Tabela-Resumo Técnica

| Plataforma | Tipo de Acesso | Endpoint Principal | Autenticação | Documentação Oficial | Custo / Limitação Relevante |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Remotive** | API JSON (pública) | `GET https://remotive.com/api/remote-jobs` | Nenhuma (gratuita) | `https://remotive.com/api-documentation` | Gratuita, porém **altamente limitada**: máx. ~4 req/dia; dados com atraso de 24h. API privada paga sob consulta (histórico indica ordem de US$ 5.000/mês). |
| **Remote OK** | API JSON (pública) | `GET https://remoteok.com/api` | Nenhuma (gratuita) | Não possui página de doc dedicada; o “contrato” está no retorno da API e em clientes não oficiais. | Gratuita, mas retorna apenas as **100 vagas mais recentes**. Exige **backlink** para a vaga original no Remote OK; caso contrário, o acesso pode ser suspenso. |
| **We Work Remotely** | **RSS (XML)** | `GET https://weworkremotely.com/remote-jobs.rss` | Nenhuma (gratuita) | Não há documentação de API JSON; o feed RSS é o canal público. | Gratuita. Não é JSON; exige *parsing* de XML. Para obter JSON, é preciso usar *scrapers* de terceiros (ex.: Apify, custo a partir de ~US$ 0,70/1.000 vagas). |
| **Himalayas** | API JSON (pública) | `GET https://himalayas.app/jobs/api` (browse) e `GET /jobs/api/search` (busca filtrada) | Nenhuma obrigatória; **chave gratuita opcional** para limites maiores. | `https://himalayas.app/docs/remote-jobs-api` | Gratuita. Sem chave, o limite é compartilhado publicamente. Com chave gratuita: **60 req/min** e **10.000 req/dia**. |
| **Jobicy** | API JSON (pública) | `GET https://jobicy.com/api/v2/remote-jobs` (ou `https://jobicy.com/api/jobs`) | Nenhuma | `https://jobi.cy/apidocs` (e repositório com OpenAPI 3.1) | Gratuita. Retorna até **200 vagas por página**, com paginação por cursor, limitado a anúncios dos **últimos 7 dias**. Exige atribuição (link para a Jobicy). |
| **Working Nomads** | API JSON (não documentada oficialmente) | `GET https://www.workingnomads.com/jobsapi/_search` | Nenhuma (aparente) | Não há documentação pública oficial; o endpoint parece ser uma API interna (Elasticsearch) exposta. | Gratuita, porém **sem garantia de estabilidade ou termos de uso**. Exige engenharia reversa e monitoramento constante. |

---

### 🔍 Detalhamento por Plataforma

#### 1. Remotive
- **URL da documentação oficial:** `https://remotive.com/api-documentation` (redireciona para o repositório no GitHub, que contém os termos e parâmetros).
- **URL principal de chamada:** `GET https://remotive.com/api/remote-jobs`. Aceita parâmetros opcionais como `category`, `company_name`, `search` e `limit`.
- **Modelo de autenticação:** Nenhum. A API é aberta, mas o uso está condicionado a termos estritos.
- **Bibliotecas oficiais:** Não há biblioteca oficial mantida pela Remotive. Existem clientes comunitários em Go, como o pacote `remotive` (`pkg.go.dev`), mas não são mantidos pela plataforma.
- **Custo e limitações:** A API pública é **gratuita**, mas a própria Remotive recomenda **no máximo 4 requisições por dia** e bloqueia mais de 2 requisições por minuto. Os dados têm **atraso de 24 horas**. Para uso intensivo, existe uma **API privada paga** (sob consulta via e-mail); o valor não é público, mas o histórico do mercado indica um patamar na casa de **US$ 5.000/mês**.
- **Observação para o CP6:** Dado o limite baixíssimo, o *scraper* da Remotive deve fazer **uma única chamada por execução de discovery** e não deve tentar paginar. O atraso de 24h é aceitável para um copiloto que roda sob demanda ou a cada 6h.

#### 2. Remote OK
- **URL da documentação oficial:** Não há uma página de documentação formal. O “contrato” de uso está implícito no retorno da API e em clientes não oficiais. A referência mais próxima é o endpoint em si.
- **URL principal de chamada:** `GET https://remoteok.com/api` (ou `https://remoteok.io/api`).
- **Modelo de autenticação:** Nenhum. A API é um documento JSON público.
- **Bibliotecas oficiais:** Não há biblioteca oficial em Go. Existem clientes comunitários em Ruby e outras linguagens, mas nenhum mantido pela Remote OK.
- **Custo e limitações:** Gratuita, mas com duas restrições críticas: (1) retorna **apenas as 100 vagas mais recentes** (uma “janela” fixa); (2) os termos exigem um **link de volta (backlink)** para a vaga original no Remote OK, sob pena de suspensão do acesso.
- **Observação para o CP6:** O *scraper* deve aceitar a limitação das 100 vagas e garantir que a UI (ou o modelo de dados) preserve a URL original da Remote OK para cumprir o requisito de atribuição.

#### 3. We Work Remotely
- **URL da documentação oficial:** A plataforma **não oferece uma API JSON pública**. O canal público é o **feed RSS**: `https://weworkremotely.com/remote-jobs.rss`.
- **URL principal de chamada:** O próprio feed RSS acima.
- **Modelo de autenticação:** Nenhum.
- **Bibliotecas oficiais:** Não se aplica (RSS). Para *parsing* de XML em Go, a biblioteca padrão `encoding/xml` é suficiente.
- **Custo e limitações:** O RSS é gratuito. A limitação é o formato (XML) e a ausência de parâmetros de filtragem ou paginação. Para obter os dados em JSON, a alternativa é usar serviços de *scraping* de terceiros (ex.: Apify), com custo a partir de **~US$ 0,70 por 1.000 vagas**.
- **Observação para o CP6:** Como o projeto já adotou `golang.org/x/net/html` para HTML, o RSS pode ser parseado com `encoding/xml` sem adicionar dependências. É o caso mais simples de “fonte que não é API JSON”.

#### 4. Himalayas
- **URL da documentação oficial:** `https://himalayas.app/docs/remote-jobs-api` — referência completa com esquema de resposta, paginação, limites e exemplos em Python/Node.js.
- **URL principal de chamada:**
  - **Browse (feed completo com paginação):** `GET https://himalayas.app/jobs/api?limit=20`
  - **Search (filtros):** `GET https://himalayas.app/jobs/api/search?keyword=...&country=...`
- **Modelo de autenticação:** **Opcional**. Sem chave, a API funciona, mas compartilha uma cota pública. Com uma **chave gratuita** (obtida em `/signup/developer`), os limites sobem para **60 requisições por minuto** e **10.000 por dia** (compartilhadas entre as chaves da conta).
- **Bibliotecas oficiais:** Não há biblioteca oficial em Go, mas a API é REST/JSON simples, sem necessidade de SDK.
- **Custo e limitações:** Gratuita dentro dos limites. O catálogo é profundo (mais de 100 mil vagas) e os dados são estruturados (salário, fuso horário, restrições geográficas).
- **Observação para o CP6:** É a fonte mais robusta para descoberta em volume. Recomenda-se obter a chave gratuita para evitar o rateio da cota pública e permitir execuções mais frequentes.

#### 5. Jobicy
- **URL da documentação oficial:** `https://jobi.cy/apidocs` (documentação interativa) e o repositório `github.com/Jobicy/remote-jobs-api`, que inclui a especificação **OpenAPI 3.1**.
- **URL principal de chamada:** `GET https://jobicy.com/api/v2/remote-jobs` (ou `https://jobicy.com/api/jobs`). Suporta `count` (até 200), `geo`, `industry`, entre outros.
- **Modelo de autenticação:** Nenhuma.
- **Bibliotecas oficiais:** Existe um pacote comunitário `jobicy` no `pkg.go.dev`, mas não é mantido pela Jobicy. A especificação OpenAPI permite gerar clientes automaticamente, se necessário.
- **Custo e limitações:** Gratuita. Retorna apenas anúncios dos **últimos 7 dias**, com paginação por cursor. Exige atribuição clara à Jobicy e que os botões de candidatura apontem para a URL original.
- **Observação para o CP6:** A paginação por cursor é útil para limitar o custo computacional. O filtro de 7 dias é compatível com um ticker de 6h ou execuções sob demanda.

#### 6. Working Nomads
- **URL da documentação oficial:** **Não há documentação pública oficial.** O endpoint `https://www.workingnomads.com/jobsapi/_search` parece ser uma API interna (Elasticsearch) exposta acidentalmente ou sem contrato público.
- **URL principal de chamada:** O endpoint acima, que retorna um JSON no formato de resposta do Elasticsearch (`hits.hits[]._source`).
- **Modelo de autenticação:** Nenhuma aparente.
- **Bibliotecas oficiais:** Nenhuma.
- **Custo e limitações:** Gratuita, mas **sem garantias de estabilidade, termos de uso ou suporte**. Qualquer mudança na infraestrutura da Working Nomads pode quebrar o *scraper* sem aviso.
- **Observação para o CP6:** Deve ser tratada como uma fonte de **baixa prioridade** ou “experimental”. Se for incluída, o agregador precisa ser tolerante a falhas (o CP6 já prevê isso) e o *scraper* deve ser simples, sem depender de parâmetros complexos.

---

### 🧩 Considerações Diretas para o Checkpoint 6 (openJob)

1.  **Prioridade de implementação:**
    - **Alta:** Himalayas (catálogo profundo, API estável, chave gratuita) e Remotive (API simples, embora limitada).
    - **Média:** Jobicy (boa paginação, OpenAPI) e Remote OK (fácil, mas janela de 100 vagas).
    - **Baixa / opcional:** We Work Remotely (RSS) e Working Nomads (API não documentada).

2.  **Estratégia de *parsing*:**
    - Para as APIs JSON (Remotive, Remote OK, Himalayas, Jobicy), use `encoding/json` da biblioteca padrão.
    - Para o RSS da We Work Remotely, use `encoding/xml`.
    - O ADR 0006 já definiu `golang.org/x/net/html` para HTML; nenhuma das fontes acima exige HTML, a menos que se opte por *scraping* de páginas de vaga individuais (o que não é o foco do discovery).

3.  **Rate limiting e tolerância a falhas:**
    - Implemente um *rate limiter* por fonte (`golang.org/x/time/rate`), com limites diferentes: Remotive ~1 req/dia (ou menos), Himalayas 60 req/min (com chave), Jobicy e Remote OK sem limites explícitos, mas com bom senso.
    - O agregador deve isolar falhas: se uma fonte retornar erro (ex.: Remotive bloqueou por excesso de requisições), as demais devem continuar.

4.  **Atribuição e *backlinks*:**
    - Remotive exige menção à fonte e *link* para a URL original; Remote OK exige *backlink*; Jobicy exige atribuição. O modelo de dados de `jobs` deve armazenar a URL canônica e a fonte, e a UI (ou o e-mail de notificação) deve exibir a atribuição.

5.  **Custo:**
    - Com as fontes gratuitas (Himalayas, Remotive, Jobicy, Remote OK, WWR RSS), é possível cobrir a descoberta sem custo adicional. A única exceção seria se o volume exigisse a API privada da Remotive (inviável para um projeto local) ou *scrapers* pagos para a WWR.

---

### 💎 Conclusão

A lista expandida confirma que o openJob pode construir o Checkpoint 6 **sem custos de API**, desde que respeite os limites da Remotive e a janela de 100 vagas da Remote OK. A combinação **Himalayas (com chave gratuita) + Remotive (1×/dia) + Jobicy (paginado) + Remote OK (janela recente) + WWR (RSS)** oferece uma base sólida e diversificada. A Working Nomads pode ser adicionada como fonte experimental, mas não deve ser um pilar da estratégia.