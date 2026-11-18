// 生成临界阻尼弹簧的 CSS linear() 缓动串；无第三方依赖。
const presets = [
  { name: 'fast', stiffness: 1600, mass: 1, zeta: 1 },
  { name: 'standard', stiffness: 900, mass: 1, zeta: 1 },
  { name: 'slow', stiffness: 400, mass: 1, zeta: 1 },
];

function springAt(t, stiffness, mass) {
  const omega = Math.sqrt(stiffness / mass);
  return 1 - (1 + omega * t) * Math.exp(-omega * t);
}

for (const preset of presets) {
  const { name, stiffness, mass, zeta } = preset;
  if (zeta !== 1) throw new Error('仅允许临界阻尼');
  const omega = Math.sqrt(stiffness / mass);
  let duration = 0.001;
  while (1 - springAt(duration, stiffness, mass) > 0.001) duration += 0.001;
  const points = [];
  for (let i = 0; i <= 50; i += 1) {
    const progress = i / 50;
    const value = springAt(progress * duration, stiffness, mass);
    points.push(`${value.toFixed(4)} ${(progress * 100).toFixed(0)}%`);
  }
  console.log(`--spring-${name}: linear(${points.join(', ')});`);
  console.log(`/* ${name}: stiffness=${stiffness}, mass=${mass}, ζ=${zeta}, 时长=${Math.round(duration * 1000)}ms */`);
}
