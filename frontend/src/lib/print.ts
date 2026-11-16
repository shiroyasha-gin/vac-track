/**
 * Print a single report node without the application shell.
 *
 * The node is cloned into a dedicated print sheet. Print styles hide the
 * application root while the sheet is active, so page orientation, headers and
 * page breaks apply to the report only.
 */
export function printReportNode(
  node: HTMLElement | null,
  options: { title: string; orientation?: 'portrait' | 'landscape' },
) {
  if (!node) {
    return;
  }

  const previousTitle = document.title;
  const sheet = document.createElement('div');
  sheet.className = 'print-sheet print-report';
  sheet.dir = 'rtl';

  const clone = node.cloneNode(true) as HTMLElement;
  clone.querySelectorAll('.no-print, [data-print-hide]').forEach((element) => element.remove());
  sheet.appendChild(clone);

  const pageStyle = document.createElement('style');
  pageStyle.textContent = `@page { size: A4 ${options.orientation ?? 'portrait'}; margin: 12mm 10mm; }`;

  document.body.appendChild(sheet);
  document.head.appendChild(pageStyle);
  document.body.classList.add('printing-sheet');
  document.title = options.title;

  try {
    window.print();
  } finally {
    document.body.classList.remove('printing-sheet');
    sheet.remove();
    pageStyle.remove();
    document.title = previousTitle;
  }
}
