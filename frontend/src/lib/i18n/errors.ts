/** Mapeamento de erros de API (CP13 §4.3 / ADR 0021). */
import { hasMessage, t } from './t.js';

export interface APIErrorBody {
	code?: string;
	params?: Record<string, string | number | boolean | null>;
	message?: string;
	error?: string;
}

/** Extrai o corpo JSON de uma resposta com segurança. */
export async function readErrorBody(res: Response): Promise<APIErrorBody> {
	try {
		return (await res.json()) as APIErrorBody;
	} catch {
		return {};
	}
}

/**
 * Texto amigável para o usuário: `code` conhecido → mensagem i18n;
 * `code` desconhecido (ou ausente) → `message` do backend (fallback EN),
 * depois `error`, por fim uma mensagem genérica traduzida.
 */
export function errorText(body: APIErrorBody | null | undefined, fallback?: string): string {
	if (!body) return fallback ?? t('errors.unknown');
	if (body.code) {
		const key = `errors.${body.code}`;
		if (hasMessage(key)) return t(key, body.params);
		// code desconhecido: fallback seguro em EN enviado pelo backend.
		return body.message || fallback || t('errors.unknown');
	}
	// Sem `code`: usa a mensagem traduzida pelo frontend, depois legados.
	return fallback || body.message || body.error || t('errors.unknown');
}

/** Erro HTTP com code + params, ou fallback em TEXT genérico traduzido. */
export function httpError(res: Response, body: APIErrorBody, fallbackKey: string): string {
	if (body.code && hasMessage(`errors.${body.code}`)) return errorText(body);
	return t(fallbackKey, { status: res.status });
}
