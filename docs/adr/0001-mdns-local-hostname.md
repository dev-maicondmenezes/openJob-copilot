# ADR-0001 — Hostname local via mDNS (`openjob.local`)

- **Status:** aceito
- **Data:** 2026-11-13
- **Contexto:** spec §4/§5 exige acesso pela rede local via **URL amigável** (não IP), do navegador do celular. Caso de uso central: revisar vagas no sofá pelo celular (§6).
- **Decisão:** usar sidecar mDNS (`skhmt/mdns`, `network_mode: host`, `HOST=openjob.local`) no `docker-compose.yml`. Fallback documentado: dnsmasq no host (`scripts/setup-dns.sh dnsmasq`); último recurso: IP direto.
- **Alternativas descartadas:**
  - *Domínio real + cloudflare tunnel*: custo, dependência de internet externa — contraria local-first.
  - *IP direto*: funciona, mas violate o requisito de URL amigável e é ruim de digitar no celular.
  - *Servidor DNS na rede*: exige acesso ao roteador; inviável para "roda no notebook do usuário".
- **Riscos conhecidos:** `.local` no macOS/Windows depende do Bonjour do host; redes corporativas às vezes bloqueiam mDNS. Mitigado pelo fallback por IP documentado no README.
- **Consequência:** sem custo de domínio (R$ 0, §11); sem mudanças no host no Linux (nativo). Revisar se mDNS se mostrar frágil em redes específicas durante o uso real (CP8).
