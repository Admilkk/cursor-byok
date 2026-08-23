import { ArcElement, Chart as ChartJS, Tooltip, type ChartOptions, type ScriptableContext } from "chart.js";
import { useMemo } from "react";
import { Doughnut } from "react-chartjs-2";
import styles from "./CacheHitRateChart.module.scss";

ChartJS.register(ArcElement, Tooltip);

type SegmentRadius = number | {
  outerStart: number;
  outerEnd: number;
  innerStart: number;
  innerEnd: number;
};

function chartColor(context: ScriptableContext<"doughnut">) {
  const styles = getComputedStyle(context.chart.canvas);
  const variable = context.dataIndex === 0 ? "--cache-hit-value-color" : "--cache-hit-track-color";
  return styles.getPropertyValue(variable).trim();
}

function segmentBorderRadius(percentage: number, dataIndex: number): SegmentRadius {
  const radius = 5;

  if (percentage <= 0) {
    return dataIndex === 1
      ? { outerStart: radius, outerEnd: radius, innerStart: radius, innerEnd: radius }
      : 0;
  }

  if (percentage >= 100) {
    return dataIndex === 0
      ? { outerStart: radius, outerEnd: radius, innerStart: radius, innerEnd: radius }
      : 0;
  }

  return dataIndex === 0
    ? { outerStart: radius, outerEnd: 0, innerStart: radius, innerEnd: 0 }
    : { outerStart: 0, outerEnd: radius, innerStart: 0, innerEnd: radius };
}

const options: ChartOptions<"doughnut"> = {
  responsive: true,
  maintainAspectRatio: false,
  cutout: "82%",
  rotation: -90,
  circumference: 180,
  animation: { duration: 450 },
  events: [],
  plugins: {
    legend: { display: false },
    tooltip: { enabled: false },
  },
};

export function CacheHitRateChart({ rate }: { rate: number }) {
  const finiteRate = Number.isFinite(rate) ? rate : 0;
  const percentage = Math.max(0, Math.min(100, finiteRate * 100));
  const label = Number.isFinite(rate) ? `${percentage.toFixed(2)}%` : "--";
  const data = useMemo(() => ({
    labels: [t("命中"), t("未命中")],
    datasets: [{
      data: [percentage, Math.max(0, 100 - percentage)],
      backgroundColor: chartColor,
      borderWidth: 0,
      hoverBorderWidth: 0,
      selfJoin: false,
      borderRadius: (context: ScriptableContext<"doughnut">) => segmentBorderRadius(percentage, context.dataIndex),
    }],
  }), [percentage]);

  return <div className={styles.root} role="img" aria-label={t("缓存命中率 {rate}", { rate: label })}>
    <Doughnut className={styles.canvas} data={data} options={options} />
    <div className={styles.label}>{label}</div>
  </div>;
}
