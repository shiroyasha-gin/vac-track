'use strict';

(function () {
  const el = {
    loading: document.getElementById('loading'),
    error: document.getElementById('error'),
    errorMessage: document.getElementById('error-message'),
    retry: document.getElementById('retry'),
    empty: document.getElementById('empty'),
    content: document.getElementById('content'),
    monthLine: document.getElementById('month-line'),
    monthLabel: document.getElementById('month-label'),
    generatedLabel: document.getElementById('generated-label'),
    vaccines: document.getElementById('stat-vaccines'),
    opening: document.getElementById('stat-opening'),
    received: document.getElementById('stat-received'),
    usage: document.getElementById('stat-usage'),
    remaining: document.getElementById('stat-remaining'),
    rows: document.getElementById('vaccine-rows'),
    tableEmpty: document.getElementById('table-empty'),
    aggregation: document.getElementById('aggregation-note'),
  };

  const formatter = new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 });
  const monthFormatter = new Intl.DateTimeFormat(undefined, { month: 'long', year: 'numeric' });
  const timestampFormatter = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' });

  function show(panel) {
    for (const section of [el.loading, el.error, el.empty, el.content]) {
      section.hidden = section !== panel;
    }
  }

  function integer(value) {
    return Number.isFinite(value) ? formatter.format(value) : '—';
  }

  function monthText(month) {
    if (typeof month !== 'string' || !/^\d{4}-\d{2}$/.test(month)) {
      return month || 'Unknown month';
    }
    const [year, monthNumber] = month.split('-').map(Number);
    return monthFormatter.format(new Date(Date.UTC(year, monthNumber - 1, 1)));
  }

  function timestampText(value) {
    if (typeof value !== 'string' || value === '') {
      return '';
    }
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? '' : timestampFormatter.format(parsed);
  }

  function cell(text, className) {
    const td = document.createElement('td');
    td.textContent = text;
    if (className) {
      td.className = className;
    }
    return td;
  }

  function render(snapshot) {
    if (!snapshot || typeof snapshot !== 'object') {
      throw new Error('The public service returned an unexpected snapshot.');
    }
    if (typeof snapshot.month !== 'string' || !snapshot.totals || !Array.isArray(snapshot.vaccines)) {
      throw new Error('The public service returned an unexpected snapshot.');
    }

    const totals = snapshot.totals;
    el.monthLabel.textContent = monthText(snapshot.month);
    const generated = timestampText(snapshot.generatedAt);
    el.generatedLabel.textContent = generated ? 'generated ' + generated : 'monthly aggregate';
    el.monthLine.hidden = false;

    el.vaccines.textContent = integer(totals.vaccineCount);
    el.opening.textContent = integer(totals.openingBalance);
    el.received.textContent = integer(totals.stockReceived);
    el.usage.textContent = integer(totals.totalUsage);
    el.remaining.textContent = integer(totals.remaining);

    el.rows.replaceChildren();
    for (const vaccine of snapshot.vaccines) {
      if (!vaccine || typeof vaccine.name !== 'string') {
        continue;
      }
      const tr = document.createElement('tr');

      const nameCell = document.createElement('td');
      const name = document.createElement('span');
      name.className = 'vaccine-name';
      name.textContent = vaccine.name;
      const unit = document.createElement('span');
      unit.className = 'vaccine-unit';
      unit.textContent = typeof vaccine.unit === 'string' ? vaccine.unit : '';
      nameCell.append(name, unit);

      tr.append(
        nameCell,
        cell(typeof vaccine.code === 'string' && vaccine.code ? vaccine.code : '—', 'muted'),
        cell(integer(vaccine.openingBalance), 'num'),
        cell(integer(vaccine.stockReceived), 'num'),
        cell(integer(vaccine.totalUsage), 'num usage-value'),
        cell(integer(vaccine.remaining), 'num'),
      );
      el.rows.append(tr);
    }

    el.tableEmpty.hidden = el.rows.childElementCount > 0;
    const aggregation =
      typeof snapshot.aggregation === 'string' && snapshot.aggregation
        ? 'Aggregation: ' + snapshot.aggregation + '.'
        : '';
    el.aggregation.textContent = aggregation;

    document.title = monthText(snapshot.month) + ' · Public immunization statistics';
    show(el.content);
  }

  async function load() {
    show(el.loading);
    el.monthLine.hidden = true;

    try {
      const response = await fetch('/api/public/snapshot', {
        method: 'GET',
        headers: { Accept: 'application/json' },
        cache: 'no-store',
      });

      if (response.status === 404) {
        show(el.empty);
        return;
      }
      if (!response.ok) {
        throw new Error('The public service returned HTTP ' + response.status + '.');
      }

      const snapshot = await response.json();
      render(snapshot);
    } catch (error) {
      el.errorMessage.textContent =
        error instanceof Error && error.message
          ? error.message
          : 'The public service did not respond. Please try again shortly.';
      show(el.error);
    }
  }

  el.retry.addEventListener('click', load);
  load();
})();
