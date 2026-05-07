type OutputHighlighter = {
  codeToHtml(code: string, options: { lang: "ansi"; theme: "vitesse-dark" }): string;
};

let outputHighlighter: Promise<OutputHighlighter> | null = null;

export async function highlightAnsiOutput(code: string) {
  outputHighlighter ??= Promise.all([
    import("@shikijs/themes/vitesse-dark"),
    import("shiki/core"),
    import("shiki/engine/javascript"),
  ]).then(([theme, core, engine]) =>
    core.createHighlighterCore({
      themes: [theme.default],
      langs: [],
      engine: engine.createJavaScriptRegexEngine(),
    }),
  );

  const highlighter = await outputHighlighter;

  return highlighter.codeToHtml(code, {
    lang: "ansi",
    theme: "vitesse-dark",
  });
}
