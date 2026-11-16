interface StatCardProps {
  label: string;
  value: number | string;
  sub?:  string;
  color?: "blue" | "yellow" | "red" | "green";
}

const colors = {
  blue:   "text-blue-400",
  yellow: "text-yellow-400",
  red:    "text-red-400",
  green:  "text-green-400",
};

export function StatCard({ label, value, sub, color = "blue" }: StatCardProps) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
      <p className="text-xs text-gray-500 uppercase tracking-wider">{label}</p>
      <p className={`text-3xl font-semibold mt-1 ${colors[color]}`}>{value}</p>
      {sub && <p className="text-xs text-gray-500 mt-1">{sub}</p>}
    </div>
  );
}
