import { t } from '#lib/i18n/t.js';

export interface Job {
	id: number;
	title: string;
	company: string;
	url: string;
	description: string;
	source: string;
	score_fit: number | null;
	score_reason: string;
	status: string;
	created_at: string;
	external_id?: string;
	matched_keywords?: string[];
	matched_resume_terms?: string[];
}

export function statusLabel(status: string): string {
	switch (status) {
		case 'new': return t('jobs.status_label.new');
		case 'approved': return t('jobs.status_label.approved');
		case 'discarded': return t('jobs.status_label.discarded');
		default: return status;
	}
}

export const SOURCE_LABELS: Record<string, string> = {
	himalayas: 'Himalayas',
	remotive: 'Remotive',
	jobicy: 'Jobicy',
	remoteok: 'Remote OK',
	manual: 'Manual'
};

export function scoreTone(score: number): 'success' | 'warning' | 'error' {
	if (score >= 75) return 'success';
	if (score >= 50) return 'warning';
	return 'error';
}
