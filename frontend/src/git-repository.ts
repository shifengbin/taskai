export function gitRepositoryPath(repository: string): string {
	const normalized = repository.trim()
	if (!normalized) {
		return ''
	}
	let path = ''
	if (normalized.includes('://')) {
		try {
			const parsed = new URL(normalized)
			if (!['http:', 'https:', 'ssh:'].includes(parsed.protocol) || !parsed.hostname || parsed.search || parsed.hash) {
				return ''
			}
			path = parsed.pathname
		} catch {
			return ''
		}
	} else {
		const separator = normalized.indexOf(':')
		if (separator <= 0 || normalized.slice(0, separator).includes('/')) {
			return ''
		}
		path = normalized.slice(separator + 1)
	}
	const segments = path.split('/').map((segment) => segment.trim()).filter(Boolean)
	if (segments.length < 2) {
		return ''
	}
	const last = segments[segments.length - 1]
	segments[segments.length - 1] = last.toLocaleLowerCase().endsWith('.git') ? last.slice(0, -4) : last
	return segments[segments.length - 1] ? segments.join('/') : ''
}
