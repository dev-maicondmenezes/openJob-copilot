// diff.ts implements line-by-line comparison (unified and side-by-side)
// using Longest Common Subsequence (LCS).

export type DiffType = 'same' | 'added' | 'removed';

export interface DiffLine {
	type: DiffType;
	text: string;
	baseLineNo?: number;
	tailoredLineNo?: number;
}

export interface SideBySideRow {
	left?: { lineNo: number; text: string; type: 'same' | 'removed' };
	right?: { lineNo: number; text: string; type: 'same' | 'added' };
}

export function computeUnifiedDiff(base: string, tailored: string): DiffLine[] {
	const a = base ? base.split('\n') : [];
	const b = tailored ? tailored.split('\n') : [];

	const n = a.length;
	const m = b.length;

	// DP table for LCS
	const dp: number[][] = Array.from({ length: n + 1 }, () => new Array(m + 1).fill(0));

	for (let i = 0; i < n; i++) {
		for (let j = 0; j < m; j++) {
			if (a[i] === b[j]) {
				dp[i + 1][j + 1] = dp[i][j] + 1;
			} else {
				dp[i + 1][j + 1] = Math.max(dp[i + 1][j], dp[i][j + 1]);
			}
		}
	}

	// Backtrack to build diff
	const result: DiffLine[] = [];
	let i = n;
	let j = m;

	const temp: DiffLine[] = [];

	while (i > 0 || j > 0) {
		if (i > 0 && j > 0 && a[i - 1] === b[j - 1]) {
			temp.push({
				type: 'same',
				text: a[i - 1],
				baseLineNo: i,
				tailoredLineNo: j
			});
			i--;
			j--;
		} else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
			temp.push({
				type: 'added',
				text: b[j - 1],
				tailoredLineNo: j
			});
			j--;
		} else if (i > 0) {
			temp.push({
				type: 'removed',
				text: a[i - 1],
				baseLineNo: i
			});
			i--;
		}
	}

	temp.reverse();
	return temp;
}

export function computeSideBySideDiff(base: string, tailored: string): SideBySideRow[] {
	const diff = computeUnifiedDiff(base, tailored);
	const rows: SideBySideRow[] = [];

	let idx = 0;
	while (idx < diff.length) {
		const item = diff[idx];

		if (item.type === 'same') {
			rows.push({
				left: { lineNo: item.baseLineNo!, text: item.text, type: 'same' },
				right: { lineNo: item.tailoredLineNo!, text: item.text, type: 'same' }
			});
			idx++;
		} else if (item.type === 'removed') {
			// Check if next item is added (paired replacement)
			if (idx + 1 < diff.length && diff[idx + 1].type === 'added') {
				const next = diff[idx + 1];
				rows.push({
					left: { lineNo: item.baseLineNo!, text: item.text, type: 'removed' },
					right: { lineNo: next.tailoredLineNo!, text: next.text, type: 'added' }
				});
				idx += 2;
			} else {
				rows.push({
					left: { lineNo: item.baseLineNo!, text: item.text, type: 'removed' }
				});
				idx++;
			}
		} else {
			// added
			rows.push({
				right: { lineNo: item.tailoredLineNo!, text: item.text, type: 'added' }
			});
			idx++;
		}
	}

	return rows;
}
