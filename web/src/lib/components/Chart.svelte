<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import uPlot from 'uplot';

  let {
    data = [],
    field,
    field2 = '',
    total = '',
    unit = '',
    color = '#58a6ff',
    color2 = '#f85149',
    label = '',
    label2 = '',
    height = 180,
  }: {
    data: any[];
    field: string;
    field2?: string;
    total?: string;
    unit?: string;
    color?: string;
    color2?: string;
    label?: string;
    label2?: string;
    height?: number;
  } = $props();

  let el: HTMLDivElement | undefined = $state();
  let chart: uPlot | null = null;
  let pendingBuild: ReturnType<typeof setTimeout> | null = null;

  function fmtVal(v: number | null | undefined): string {
    if (v == null) return '--';
    if (unit === 'bytes' || unit === 'bytes/s') {
      const abs = Math.abs(v);
      if (abs >= 1073741824) return (v / 1073741824).toFixed(1) + ' GB';
      if (abs >= 1048576) return (v / 1048576).toFixed(1) + ' MB';
      if (abs >= 1024) return (v / 1024).toFixed(1) + ' KB';
      return v.toFixed(0) + ' B';
    }
    if (unit === '%') return v.toFixed(1) + '%';
    return v.toFixed(2);
  }

  function fmtAxis(_: uPlot, vals: number[]): string[] {
    return vals.map(v => {
      if (v == null) return '';
      if (unit === '%') return v.toFixed(0) + '%';
      if (unit === 'bytes' || unit === 'bytes/s') {
        const abs = Math.abs(v);
        if (abs >= 1073741824) return (v / 1073741824).toFixed(0) + 'G';
        if (abs >= 1048576) return (v / 1048576).toFixed(0) + 'M';
        if (abs >= 1024) return (v / 1024).toFixed(0) + 'K';
        return v.toFixed(0);
      }
      return v % 1 === 0 ? v.toFixed(0) : v.toFixed(1);
    });
  }

  function hexToRGBA(hex: string, alpha: number): string {
    const r = parseInt(hex.slice(1, 3), 16);
    const g = parseInt(hex.slice(3, 5), 16);
    const b = parseInt(hex.slice(5, 7), 16);
    return `rgba(${r},${g},${b},${alpha})`;
  }

  function tryBuild(retries = 5) {
    if (!el || data.length === 0) return;

    const w = el.clientWidth;
    if (w < 50) {
      if (retries > 0) {
        pendingBuild = setTimeout(() => tryBuild(retries - 1), 100);
      }
      return;
    }

    if (chart) { chart.destroy(); chart = null; }
    el.innerHTML = '';

    const timestamps = data.map((d: any) => Math.floor(new Date(d.ts).getTime() / 1000));

    const values1 = data.map((d: any) => {
      if (total && d[total] > 0) return (d[field] / d[total]) * 100;
      return d[field] ?? null;
    });

    let seriesDef: uPlot.Series[] = [
      { label: 'Time' },
      {
        label: label || field,
        stroke: color,
        width: 1.5,
        fill: hexToRGBA(color, 0.12),
        value: (_: uPlot, v: number | null) => fmtVal(v),
        points: { show: false },
      },
    ];

    let plotData: uPlot.AlignedData = [timestamps, values1];

    if (field2) {
      const values2 = data.map((d: any) => d[field2] ?? null);
      seriesDef.push({
        label: label2 || field2,
        stroke: color2,
        width: 1.5,
        fill: hexToRGBA(color2, 0.08),
        value: (_: uPlot, v: number | null) => fmtVal(v),
        points: { show: false },
      });
      plotData = [timestamps, values1, values2];
    }

    const opts: uPlot.Options = {
      width: w,
      height,
      padding: [8, 8, 0, 0],
      series: seriesDef,
      axes: [
        {
          stroke: '#8b949e88',
          ticks: { stroke: '#8b949e22', width: 1 },
          grid: { stroke: '#8b949e11', width: 1 },
          font: '10px system-ui, sans-serif',
          gap: 4,
          size: 36,
        },
        {
          stroke: '#8b949e88',
          ticks: { stroke: '#8b949e22', width: 1 },
          grid: { stroke: '#8b949e15', width: 1 },
          font: '10px system-ui, sans-serif',
          values: fmtAxis,
          gap: 4,
          size: 50,
        },
      ],
      scales: {
        x: { time: true },
        y: (total || unit === '%') ? { min: 0, max: 100 } : { auto: true },
      },
      legend: { show: false },
      cursor: {
        show: true,
        points: { show: false },
      },
    };

    chart = new uPlot(opts, plotData, el);
  }

  function scheduleBuild() {
    if (pendingBuild) clearTimeout(pendingBuild);
    pendingBuild = setTimeout(() => {
      pendingBuild = null;
      tryBuild(5);
    }, 50);
  }

  $effect(() => {
    if (el && data && data.length > 0) {
      scheduleBuild();
    }
  });

  let resizeObserver: ResizeObserver | null = null;

  onMount(() => {
    resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const w = Math.floor(entry.contentRect.width);
        if (w < 50) continue;
        if (chart) {
          chart.setSize({ width: w, height });
        } else if (data.length > 0) {
          tryBuild(3);
        }
      }
    });
    if (el) resizeObserver.observe(el);
  });

  onDestroy(() => {
    if (pendingBuild) clearTimeout(pendingBuild);
    resizeObserver?.disconnect();
    if (chart) { chart.destroy(); chart = null; }
  });
</script>

<div class="chart-wrap" bind:this={el}></div>

<style>
  .chart-wrap {
    width: 100%;
    min-height: 180px;
  }
</style>
