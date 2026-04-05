import { readFileSync, readdirSync, statSync } from 'fs';
import { resolve, join } from 'path';

const tokensCSS = readFileSync(resolve(__dirname, './tokens.css'), 'utf-8');

/**
 * Helper: extract the first top-level :root { ... } block (light theme).
 * Stops at the matching closing brace.
 */
function extractLightRootBlock(css: string): string {
  const match = css.match(/^:root\s*\{([^}]*)\}/m);
  return match ? match[1] : '';
}

/**
 * Helper: extract the dark theme :root block inside @media (prefers-color-scheme: dark).
 */
function extractDarkRootBlock(css: string): string {
  const match = css.match(
    /@media\s*\(prefers-color-scheme:\s*dark\)\s*\{[\s\S]*?:root\s*\{([\s\S]*?)\}\s*\}/
  );
  return match ? match[1] : '';
}

/**
 * Helper: parse all CSS custom property names from a block of CSS text.
 */
function parseTokenNames(block: string): string[] {
  const names: string[] = [];
  const re = /(--[\w-]+)\s*:/g;
  let m: RegExpExecArray | null;
  while ((m = re.exec(block)) !== null) {
    names.push(m[1]);
  }
  return names;
}

/**
 * Recursively collect all CSS files under a directory.
 */
function collectCSSFiles(dir: string): string[] {
  const results: string[] = [];
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) {
      results.push(...collectCSSFiles(full));
    } else if (entry.endsWith('.css')) {
      results.push(full);
    }
  }
  return results;
}

const lightBlock = extractLightRootBlock(tokensCSS);
const darkBlock = extractDarkRootBlock(tokensCSS);
const lightTokens = parseTokenNames(lightBlock);
const darkTokens = parseTokenNames(darkBlock);

// ── 1. Token existence (light theme) ──────────────────────────────────────────

describe('Token existence (light theme)', () => {
  it('should have tokens defined in :root', () => {
    expect(lightTokens.length).toBeGreaterThan(0);
  });

  const expectedPrefixes = ['--color-', '--radius-', '--shadow-', '--font-', '--space-'];

  for (const prefix of expectedPrefixes) {
    it(`should contain at least one ${prefix}* token`, () => {
      const matching = lightTokens.filter((t) => t.startsWith(prefix));
      expect(matching.length).toBeGreaterThan(0);
    });
  }

  it('all light tokens should be accessible (defined in :root block)', () => {
    for (const token of lightTokens) {
      // Each token must have a value assigned
      const re = new RegExp(`${token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*:`);
      expect(lightBlock).toMatch(re);
    }
  });
});

// ── 2. Indigo palette verification ────────────────────────────────────────────

describe('Indigo palette verification', () => {
  it('--color-primary should be Indigo-500 (#6366f1)', () => {
    expect(lightBlock).toContain('#6366f1');
  });

  it('--color-primary-hover should be Indigo-600 (#4f46e5)', () => {
    expect(lightBlock).toContain('#4f46e5');
  });

  it('should NOT contain Blue (#3b82f6) in the light :root block', () => {
    expect(lightBlock).not.toContain('#3b82f6');
  });
});

// ── 3. Dark theme existence ───────────────────────────────────────────────────

describe('Dark theme existence', () => {
  it('should contain @media (prefers-color-scheme: dark) block', () => {
    expect(tokensCSS).toMatch(/@media\s*\(prefers-color-scheme:\s*dark\)/);
  });
});

// ── 4. Dark theme token coverage ──────────────────────────────────────────────

describe('Dark theme token coverage', () => {
  const lightColorAndShadowTokens = lightTokens.filter(
    (t) => t.startsWith('--color-') || t.startsWith('--shadow-')
  );

  it('should have color/shadow tokens in the light theme to test', () => {
    expect(lightColorAndShadowTokens.length).toBeGreaterThan(0);
  });

  for (const token of lightColorAndShadowTokens) {
    it(`dark theme should override ${token}`, () => {
      expect(darkTokens).toContain(token);
    });
  }
});

// ── 5. No hardcoded hex colors in component CSS files ─────────────────────────

describe('No hardcoded hex colors in component CSS files', () => {
  const componentsDir = resolve(__dirname, './components');
  const cssFiles = collectCSSFiles(componentsDir);

  it('should find component CSS files to scan', () => {
    expect(cssFiles.length).toBeGreaterThan(0);
  });

  for (const filePath of cssFiles) {
    const shortName = filePath.replace(componentsDir + '/', '');

    it(`${shortName} should not contain hardcoded hex colors`, () => {
      const content = readFileSync(filePath, 'utf-8');
      // Match hex colors: #RGB, #RRGGBB (3, 4, 6, or 8 hex digits)
      // Allow #fff, #ffffff, #000, #000000
      const hexRe = /#(?:[0-9a-fA-F]{3,8})\b/g;
      const allowed = new Set(['#fff', '#ffffff', '#000', '#000000']);
      let match: RegExpExecArray | null;
      const violations: string[] = [];

      while ((match = hexRe.exec(content)) !== null) {
        const hex = match[0].toLowerCase();
        if (!allowed.has(hex)) {
          violations.push(match[0]);
        }
      }

      expect(
        violations,
        `Found hardcoded hex colors in ${shortName}: ${violations.join(', ')}`
      ).toEqual([]);
    });
  }
});

// ── 6. Dark theme contrast check ──────────────────────────────────────────────

describe('Dark theme contrast check', () => {
  it('light --color-primary should be Indigo-500 (#6366f1)', () => {
    const match = lightBlock.match(/--color-primary\s*:\s*(#[0-9a-fA-F]+)/);
    expect(match).not.toBeNull();
    expect(match![1]).toBe('#6366f1');
  });

  it('dark --color-primary should be Indigo-400 (#818cf8), lighter than light theme', () => {
    const match = darkBlock.match(/--color-primary\s*:\s*(#[0-9a-fA-F]+)/);
    expect(match).not.toBeNull();
    expect(match![1]).toBe('#818cf8');
  });
});

// ── 7. All new semantic tokens from issue [42] ───────────────────────────────

describe('Semantic tokens from issue #42', () => {
  const requiredTokens = [
    '--color-border-hover',
    '--color-danger',
    '--color-danger-hover',
    '--color-warning-bg',
    '--color-warning-text',
    '--color-danger-bg',
    '--color-danger-text',
    '--color-danger-text-dark',
  ];

  for (const token of requiredTokens) {
    it(`${token} should exist in light theme`, () => {
      expect(lightTokens).toContain(token);
    });

    it(`${token} should exist in dark theme`, () => {
      expect(darkTokens).toContain(token);
    });
  }
});
