<script lang="ts">
	interface Props {
		content: string;
	}

	let { content = '' }: Props = $props();

	function escapeHtml(text: string): string {
		return text
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;')
			.replace(/"/g, '&quot;')
			.replace(/'/g, '&#039;');
	}

	function formatInline(text: string): string {
		let s = escapeHtml(text);
		// Bold: **text** or __text__
		s = s.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
		s = s.replace(/__(.*?)__/g, '<strong>$1</strong>');
		// Italic: *text* or _text_
		s = s.replace(/\*(.*?)\*/g, '<em>$1</em>');
		s = s.replace(/_(.*?)_/g, '<em>$1</em>');
		// Inline code: `code`
		s = s.replace(/`([^`]+)`/g, '<code>$1</code>');
		// Links: [text](url)
		s = s.replace(/\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>');
		return s;
	}

	interface Block {
		kind: 'h1' | 'h2' | 'h3' | 'ul' | 'p';
		text?: string;
		items?: string[];
	}

	let blocks = $derived.by<Block[]>(() => {
		if (!content) return [];
		const lines = content.split('\n');
		const result: Block[] = [];
		let currentList: string[] = [];

		const flushList = () => {
			if (currentList.length > 0) {
				result.push({ kind: 'ul', items: [...currentList] });
				currentList = [];
			}
		};

		for (const rawLine of lines) {
			const line = rawLine.trim();
			if (!line) {
				flushList();
				continue;
			}

			if (line.startsWith('# ')) {
				flushList();
				result.push({ kind: 'h1', text: line.substring(2) });
			} else if (line.startsWith('## ')) {
				flushList();
				result.push({ kind: 'h2', text: line.substring(3) });
			} else if (line.startsWith('### ')) {
				flushList();
				result.push({ kind: 'h3', text: line.substring(4) });
			} else if (line.startsWith('- ') || line.startsWith('* ')) {
				currentList.push(line.substring(2));
			} else {
				flushList();
				result.push({ kind: 'p', text: line });
			}
		}

		flushList();
		return result;
	});
</script>

<div class="markdown-preview">
	{#each blocks as block}
		{#if block.kind === 'h1'}
			<h1>{@html formatInline(block.text ?? '')}</h1>
		{:else if block.kind === 'h2'}
			<h2>{@html formatInline(block.text ?? '')}</h2>
		{:else if block.kind === 'h3'}
			<h3>{@html formatInline(block.text ?? '')}</h3>
		{:else if block.kind === 'ul'}
			<ul>
				{#each block.items ?? [] as item}
					<li>{@html formatInline(item)}</li>
				{/each}
			</ul>
		{:else if block.kind === 'p'}
			<p>{@html formatInline(block.text ?? '')}</p>
		{/if}
	{/each}
</div>

<style>
	.markdown-preview {
		font-size: 0.95rem;
		line-height: 1.6;
		color: var(--text);
	}

	.markdown-preview h1 {
		font-size: 1.35rem;
		font-weight: 700;
		margin: 0.75rem 0 0.5rem;
		border-bottom: 2px solid var(--border);
		padding-bottom: 0.25rem;
	}

	.markdown-preview h2 {
		font-size: 1.15rem;
		font-weight: 600;
		margin: 1.25rem 0 0.4rem;
		color: var(--accent);
		border-bottom: 1px solid var(--border);
		padding-bottom: 0.2rem;
	}

	.markdown-preview h3 {
		font-size: 1rem;
		font-weight: 600;
		margin: 0.8rem 0 0.25rem;
	}

	.markdown-preview p {
		margin: 0.4rem 0;
	}

	.markdown-preview ul {
		margin: 0.4rem 0 0.75rem 1.25rem;
		padding: 0;
	}

	.markdown-preview li {
		margin-bottom: 0.25rem;
	}

	:global(.markdown-preview code) {
		background: #f1f5f9;
		padding: 0.15rem 0.35rem;
		border-radius: 4px;
		font-size: 0.85em;
		font-family: monospace;
	}

	:global(.markdown-preview a) {
		color: var(--accent);
		text-decoration: underline;
	}
</style>
