<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import uPlot from 'uplot';
  import 'uplot/dist/uPlot.min.css';

  export let data: any[] = [];
  export let field: string;
  export let field2: string = '';
  export let total: string = '';
  export let unit: string = '';
  export let color: string = '#58a6ff';
  export let color2: string = '#f85149';

  let el: HTMLDivElement;
  let chart: uPlot | null = null;

  function formatValue(v: number): string {
    if (unit === 'bytes' || unit === 'bytes/s') {
      if (v > 1073741824) return (v / 1073741824).toFixed(1) + ' GB';
      if (v > 1048576) return (v / 1048576).toFixed(1) + ' MB';
      if (v > 1024) return (v / 1024).toFixed(1) + ' KB';
      return v.toFixed(0) + ' B';
    }
    if (unit === '%') return v.toFixed(1) + '%';
    return v.toFixed(2);
  }

  function buildChart() {
    if (!el || data.length === 0) return;
    if (chart) { chart.destroy(); chart = null; }

    const timestamps = data.map((d: any) => new Date(d.ts).getTime() / 1000);

    let series: uPlot.Series[] = [{}];
    let plotData: uPlot.AlignedData;

    const values1 = data.map((d: any) => {
      if (total) return (d[field] / d[total]) * 100;
      return d[field] ?? 0;
    });

    if (field2) {
      const values2 = data.map((d: any) => d[field2] ?? 0);
      series.push(
        { label: field, stroke: color, width: 2, fill: color + '15' },
        { label: field2, stroke: color2, width: 2, fill: color2 + '15' }
      );
      plotData = [timestamps, values1, values2];
    } else {
      series.push(
        { label: field, stroke: color, width: 2, fill: color + '15' }
      );
      plotData = [timestamps, values1];
    }

    const opts: uPlot.Options = {
      width: el.clientWidth,
      height: 200,
      series,
      axes: [
        { stroke: '#8b949e44', grid: { stroke: '#8b949e11' } },
        {
          stroke: '#8b949e44',
          grid: { stroke: '#8b949e11' },
          values: (_: any, vals: number[]) => vals.map(v => formatValue(v)),
        },
      ],
      cursor: { show: true },
      scales: {
        y: total ? { min: 0, max: 100 } : {},
      },
    };

    chart = new uPlot(opts, plotData, el);
  }

  $: if (data.length > 0 && el) buildChart();

  onMount(() => {
    const observer = new ResizeObserver(() => {
      if (chart && el) chart.setSize({ width: el.clientWidth, height: 200 });
    });
    if (el) observer.observe(el);
    return () => observer.disconnect();
  });

  onDestroy(() => { if (chart) chart.destroy(); });
</script>

<div class="chart-container" bind:this={el}></div>

<style>
  .chart-container {
    width: 100%;
    min-height: 200px;
  }
  :global(.u-wrap) { width: 100% !important; }
</style>
