import { type PointerEvent, useState } from "react";

type DigitalEmployee3DProps = {
  active: boolean;
  thinking: boolean;
};

export function DigitalEmployee3D({ active, thinking }: DigitalEmployee3DProps) {
  const [gaze, setGaze] = useState({ x: 0, y: 0 });

  function handlePointerMove(event: PointerEvent<HTMLDivElement>) {
    const rect = event.currentTarget.getBoundingClientRect();
    const x = ((event.clientX - rect.left) / rect.width - 0.5) * 2;
    const y = ((event.clientY - rect.top) / rect.height - 0.5) * 2;
    setGaze({
      x: Math.max(-1, Math.min(1, x)),
      y: Math.max(-1, Math.min(1, y)),
    });
  }

  const eyeTransform = `translate(${gaze.x * 3.6}px ${gaze.y * 2.8}px)`;
  const headTransform = `translate(${gaze.x * 1.5}px ${gaze.y * 1}px) rotate(${gaze.x * 1.5}deg)`;

  return (
    <div
      className={`employee-avatar-stage ${active ? "avatar-active" : ""} ${thinking ? "avatar-thinking" : ""}`}
      aria-label="Digital employee avatar"
      onPointerLeave={() => setGaze({ x: 0, y: 0 })}
      onPointerMove={handlePointerMove}
    >
      <svg className="employee-avatar-svg" viewBox="0 0 180 180" role="img">
        <defs>
          <linearGradient id="avatarBackdrop" x1="20" y1="8" x2="150" y2="168" gradientUnits="userSpaceOnUse">
            <stop stopColor="#FFFFFF" />
            <stop offset="0.54" stopColor="#F2FBFF" />
            <stop offset="1" stopColor="#E6F2FF" />
          </linearGradient>
          <linearGradient id="avatarSuit" x1="53" y1="90" x2="130" y2="157" gradientUnits="userSpaceOnUse">
            <stop stopColor="#2F7DF4" />
            <stop offset="1" stopColor="#143E9A" />
          </linearGradient>
          <filter id="avatarShadow" x="-20%" y="-20%" width="140%" height="150%">
            <feDropShadow dx="0" dy="14" stdDeviation="12" floodColor="#18345E" floodOpacity="0.18" />
          </filter>
          <filter id="avatarSoftGlow" x="-40%" y="-40%" width="180%" height="180%">
            <feGaussianBlur stdDeviation="3" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        <rect x="10" y="10" width="160" height="160" rx="34" fill="url(#avatarBackdrop)" />
        <rect x="12" y="12" width="156" height="156" rx="32" fill="none" stroke="#FFFFFF" strokeWidth="2" opacity="0.8" />
        <path className="avatar-orbit" d="M37 137C57 126 123 125 145 138" fill="none" stroke="#46D8EE" strokeWidth="3" strokeLinecap="round" opacity="0.9" />
        <ellipse cx="90" cy="150" rx="47" ry="11" fill="#7DA9E8" opacity="0.2" />

        <g className="avatar-person" filter="url(#avatarShadow)">
          <path className="avatar-tail" d="M121 124C151 123 153 88 133 83C118 80 116 98 130 101" fill="none" stroke="#FFBE3D" strokeWidth="12" strokeLinecap="round" />
          <path d="M55 121C58 101 72 91 90 91C108 91 122 101 125 121L130 153H50L55 121Z" fill="url(#avatarSuit)" />
          <path d="M67 122L90 104L113 122V153H67V122Z" fill="#1D5DD6" opacity="0.65" />
          <path d="M74 102L90 117L106 102" fill="none" stroke="#EAF6FF" strokeWidth="8" strokeLinecap="round" strokeLinejoin="round" opacity="0.95" />
          <circle className="avatar-core" cx="90" cy="126" r="7" fill="#38D99D" filter="url(#avatarSoftGlow)" />
          <path className="avatar-paw-left" d="M62 129C50 126 43 119 40 109" stroke="#236BE6" strokeWidth="9" strokeLinecap="round" />
          <path className="avatar-paw-right" d="M118 129C130 126 137 119 140 109" stroke="#236BE6" strokeWidth="9" strokeLinecap="round" />
          <circle cx="39" cy="107" r="6" fill="#FFE0A3" />
          <circle cx="141" cy="107" r="6" fill="#FFE0A3" />
          <ellipse cx="73" cy="154" rx="13" ry="6" fill="#1647AA" opacity="0.9" />
          <ellipse cx="107" cy="154" rx="13" ry="6" fill="#1647AA" opacity="0.9" />

          <g className="avatar-head" style={{ transform: headTransform }}>
            <image href="/assets/noto-cat-face.svg" x="35" y="21" width="110" height="110" />
            <path d="M49 70H42C38 70 35 74 35 79V88C35 93 38 97 42 97H49" fill="#10233F" opacity="0.94" />
            <path d="M131 70H138C142 70 145 74 145 79V88C145 93 142 97 138 97H131" fill="#10233F" opacity="0.94" />
            <path d="M135 93C143 98 147 103 148 110" fill="none" stroke="#10233F" strokeWidth="4" strokeLinecap="round" />
            <circle cx="149" cy="112" r="4.5" fill="#39DDA1" />
          </g>
          <g className="avatar-eye-glints" style={{ transform: eyeTransform }}>
            <circle cx="75" cy="73" r="2.4" fill="#FFFFFF" opacity="0.95" />
            <circle cx="108" cy="73" r="2.4" fill="#FFFFFF" opacity="0.95" />
            <circle cx="73" cy="76" r="1.2" fill="#10233F" opacity="0.45" />
            <circle cx="106" cy="76" r="1.2" fill="#10233F" opacity="0.45" />
          </g>
        </g>

        <path className="avatar-signal" d="M43 36C55 22 72 17 90 17C108 17 125 22 137 36" fill="none" stroke="#38D99D" strokeWidth="4" strokeLinecap="round" opacity="0.85" />
      </svg>
    </div>
  );
}
