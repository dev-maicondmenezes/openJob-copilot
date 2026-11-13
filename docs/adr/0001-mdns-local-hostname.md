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

---

## Adendo (2026-11-13) — aprendidos da validação real (VirtualBox + LAN Wi-Fi)

Validação feita em VM VirtualBox (Ubuntu guest) com o celular na mesma rede:

1. **VirtualBox NAT ≠ LAN.** Com o adaptador em modo NAT (`10.0.2.15`), a VM é invisível para a rede local. Solução: **Adaptador em modo Bridge** no VirtualBox. Com NAT, o fallback é redirecionamento de porta (host 8080 → guest 8080).
2. **Sidecar `skhmt/mdns`:** respondeu às consultas (`query:`/`responding:` nos logs), mas a resposta **não chegava ao cliente na própria máquina** (loopback multicast desabilitado no responder). Para clientes externos o anúncio funcionou (Windows resolveu). Conflito potencial de porta 5353 com o `avahi-daemon` nativo do guest.
3. **Caminho nativo no Linux guest (preferido):** publicar alias no **arquivo único** `/etc/avahi/hosts` (não é diretório) com `192.168.15.118 openjob.local` + `systemctl restart avahi-daemon`. Nesse cenário o sidecar é redundante — manter **um único respondente** mDNS por rede. O sidecar fica no compose para os cenários Mac/Windows (sem avahi no guest).
4. **Loopback local:** `avahi-resolve -n openjob.local` pode dar timeout na própria máquina mesmo com o anúncio sadio na rede. Cobertura local garantida por `/etc/hosts` (`files` vem primeiro no `nsswitch.conf`).
5. **⚠️ Android/Chromium não resolve `.local`** — limitação do Chromium (mDNS lookup não implementado no resolver de hostname do navegador). Afeta Chrome/Android independente da qualidade do anúncio. **Mitigação adotada:** domínio "mágico" **`nip.io`/`sslip.io`** — `http://192.168.15.118.nip.io:8080` resolve via DNS público para o IP embutido no nome; funciona em qualquer navegador/dispositivo, sem infraestrutura. URL amigável preservada (não é IP cru).
6. **Hostname da máquina permanece intacto** — o alias `openjob.local` é publicado pelo avahi como registro adicional; não há necessidade de `hostnamectl hostname openjob`.
7. **Decisão final (2026-11-13):** para uso local desktop, o nome canônico do app passa a ser **`openjob.copilot`**, resolvido por **hosts-file alias** (`/etc/hosts` na VM; `C:\Windows\System32\drivers\etc\hosts` no Windows host — zero serviço extra, zero config de rede). `OPENJOB_HOSTNAME=openjob.copilot` no `.env`. O mDNS `openjob.local` (via `/etc/avahi/hosts`) permanece como conveniência para clientes LAN que suportam mDNS. Acesso pelo **celular Android fica para o app nativo planejado pós-validação (CP8+)** — nenhuma config de DNS mobile é mantida; Android pode usar o IP ou `nip.io` provisoriamente.

