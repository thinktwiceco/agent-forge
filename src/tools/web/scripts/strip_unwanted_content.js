(function() {
	// --- Signal A: SPA / framework detection ---
	const isSPA = !!(
		window.__NEXT_DATA__   ||
		window.__nuxt__        ||
		window.__remixContext  ||
		window.angular         ||
		document.querySelector('#__next, #app, [data-reactroot], [data-v-app]')
	);

	if (isSPA) {
		return { isSPA: true, skipped: true, removedCount: 0, overStripped: false, fallbackText: '' };
	}

	// --- Capture content before any DOM mutation ---
	const beforeText = (document.body.innerText || '').trim();
	const beforeLen  = beforeText.length;

	// Never remove an element that contains interactive children — the agent
	// needs links, buttons, and inputs to know what to click next.
	function hasInteractive(el) {
		return !!el.querySelector('a[href], button, input, select, textarea, [role="button"], [onclick]');
	}

	function safeRemove(el) {
		if (el.closest('main, article, [role="main"], [role="article"]')) return false;
		if (hasInteractive(el)) return false;
		el.remove();
		return true;
	}

	// --- Semantic noise selectors (static / server-rendered pages) ---
	// nav/header/footer are intentionally excluded — they hold navigation links.
	const selectors = [
		'aside', '[role="complementary"]',
		'[class*="advertisement"]', '[class*="promo"]',
		'[id*="advertisement"]', '[id*="promo"]',
		'[class*="sponsor"]', '[id*="sponsor"]',
		'[class*="cookie"]', '[id*="cookie"]',
		'[class*="consent"]', '[id*="consent"]',
		'[class*="gdpr"]', '[id*="gdpr"]',
		'[class*="share"]', '[id*="share"]',
		'[class*="comment"]', '[id*="comment"]',
		'[class*="discussion"]', '[id*="discussion"]',
		'[class*="newsletter"]', '[id*="newsletter"]',
		'[class*="breadcrumb"]', '[id*="breadcrumb"]',
		'nav[aria-label*="breadcrumb"]',
		'[class*="skip"]', '[id*="skip"]',
		'[style*="display: none"]', '[style*="display:none"]',
		'[hidden]',
	];

	let removedCount = 0;
	selectors.forEach(selector => {
		try {
			document.querySelectorAll(selector).forEach(el => {
				if (safeRemove(el)) removedCount++;
			});
		} catch (e) {}
	});

	// Pattern-based removal — deliberately conservative patterns only
	const unwantedPatterns = [
		/\bpromo\b/i, /\bsponsor\b/i, /\bbanner\b/i,
		/\bcookie\b/i, /\bconsent\b/i,
	];

	document.querySelectorAll('*').forEach(el => {
		const combined = ((el.className || '') + ' ' + (el.id || '')).toLowerCase();
		for (const pattern of unwantedPatterns) {
			if (pattern.test(combined) && safeRemove(el)) { removedCount++; break; }
		}
	});

	// --- Signal B: content ratio guard ---
	const afterLen = (document.body.innerText || '').trim().length;
	const ratio = beforeLen > 0 ? afterLen / beforeLen : 1;
	const overStripped = ratio < 0.5;

	return {
		isSPA: false, skipped: false, removedCount,
		beforeLen, afterLen, ratio,
		overStripped,
		fallbackText: overStripped ? beforeText : '',
	};
})();
