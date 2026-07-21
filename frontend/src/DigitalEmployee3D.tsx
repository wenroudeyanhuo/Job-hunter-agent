type DigitalEmployee3DProps = {
  active: boolean;
  thinking: boolean;
};

export function DigitalEmployee3D({ active, thinking }: DigitalEmployee3DProps) {
  return (
    <div className={`employee-avatar-stage ${active ? "avatar-active" : ""} ${thinking ? "avatar-thinking" : ""}`} aria-label="Digital employee avatar">
      <svg className="employee-avatar-svg" viewBox="0 0 180 180" role="img">
        <defs>
          <linearGradient id="avatarBackdrop" x1="20" y1="8" x2="150" y2="168" gradientUnits="userSpaceOnUse">
            <stop stopColor="#F9FCFF" />
            <stop offset="0.52" stopColor="#EAF4FF" />
            <stop offset="1" stopColor="#DDEBFF" />
          </linearGradient>
          <linearGradient id="avatarSuit" x1="51" y1="82" x2="132" y2="160" gradientUnits="userSpaceOnUse">
            <stop stopColor="#1F74E8" />
            <stop offset="0.62" stopColor="#1954B8" />
            <stop offset="1" stopColor="#102F72" />
          </linearGradient>
          <linearGradient id="avatarFur" x1="48" y1="26" x2="130" y2="118" gradientUnits="userSpaceOnUse">
            <stop stopColor="#FFF8EF" />
            <stop offset="0.58" stopColor="#F0D4B8" />
            <stop offset="1" stopColor="#D9A981" />
          </linearGradient>
          <linearGradient id="avatarMuzzle" x1="66" y1="78" x2="112" y2="112" gradientUnits="userSpaceOnUse">
            <stop stopColor="#FFFDF8" />
            <stop offset="1" stopColor="#F7E5D2" />
          </linearGradient>
          <linearGradient id="avatarGlow" x1="53" y1="107" x2="124" y2="152" gradientUnits="userSpaceOnUse">
            <stop stopColor="#87F7FF" />
            <stop offset="1" stopColor="#1BA977" />
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
        <path className="avatar-orbit" d="M37 137C57 126 123 125 145 138" fill="none" stroke="#46D8EE" strokeWidth="3" strokeLinecap="round" opacity="0.9" />
        <ellipse cx="90" cy="149" rx="46" ry="11" fill="#7DA9E8" opacity="0.2" />

        <g className="avatar-person" filter="url(#avatarShadow)">
          <path d="M54 124C57 99 72 86 90 86C108 86 123 99 126 124L130 153H50L54 124Z" fill="url(#avatarSuit)" />
          <path d="M63 104C72 113 80 117 90 117C100 117 109 113 117 104L124 127C115 140 103 146 90 146C77 146 65 140 56 127L63 104Z" fill="#143E8E" opacity="0.5" />
          <path d="M70 99H110L104 122H76L70 99Z" fill="#F8FBFF" opacity="0.98" />
          <path d="M80 101L90 117L100 101" fill="none" stroke="#D5E7FF" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
          <circle className="avatar-core" cx="90" cy="128" r="8" fill="#38D99D" filter="url(#avatarSoftGlow)" />
          <path d="M55 124L38 114" stroke="#173970" strokeWidth="9" strokeLinecap="round" />
          <path d="M125 124L142 114" stroke="#173970" strokeWidth="9" strokeLinecap="round" />
          <circle cx="35" cy="113" r="6" fill="#FFF8EF" />
          <circle cx="145" cy="113" r="6" fill="#FFF8EF" />

          <path d="M52 50L59 20L81 42Z" fill="url(#avatarFur)" />
          <path d="M128 50L121 20L99 42Z" fill="url(#avatarFur)" />
          <path d="M60 43L63 30L73 43Z" fill="#F4B8AE" opacity="0.78" />
          <path d="M120 43L117 30L107 43Z" fill="#F4B8AE" opacity="0.78" />
          <path d="M45 66C45 40 64 27 90 27C116 27 135 40 135 66V77C135 102 116 116 90 116C64 116 45 102 45 77V66Z" fill="url(#avatarFur)" />
          <path d="M57 58C61 45 73 38 90 38C107 38 119 45 123 58C113 53 102 50 90 50C78 50 67 53 57 58Z" fill="#FFFFFF" opacity="0.42" />
          <path d="M62 75C65 66 75 61 90 61C105 61 115 66 118 75V84C118 95 106 103 90 103C74 103 62 95 62 84V75Z" fill="rgba(255,255,255,0.42)" />
          <path className="avatar-eye-left" d="M67 73C71 70 76 70 80 73" fill="none" stroke="#102033" strokeWidth="4" strokeLinecap="round" />
          <path className="avatar-eye-right" d="M100 73C104 70 109 70 113 73" fill="none" stroke="#102033" strokeWidth="4" strokeLinecap="round" />
          <path d="M85 83L90 87L95 83" fill="none" stroke="#8A5A44" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
          <path d="M90 87V92" stroke="#8A5A44" strokeWidth="2.5" strokeLinecap="round" />
          <path d="M80 94C85 99 95 99 100 94" fill="none" stroke="#8A5A44" strokeWidth="3" strokeLinecap="round" />
          <path d="M58 86H44" stroke="#D49976" strokeWidth="2" strokeLinecap="round" opacity="0.85" />
          <path d="M59 93H45" stroke="#D49976" strokeWidth="2" strokeLinecap="round" opacity="0.72" />
          <path d="M122 86H136" stroke="#D49976" strokeWidth="2" strokeLinecap="round" opacity="0.85" />
          <path d="M121 93H135" stroke="#D49976" strokeWidth="2" strokeLinecap="round" opacity="0.72" />
          <path d="M48 70H41C37 70 34 74 34 79V86C34 91 37 95 41 95H48" fill="#10233F" />
          <path d="M132 70H139C143 70 146 74 146 79V86C146 91 143 95 139 95H132" fill="#10233F" />
          <path d="M134 91C142 96 146 101 147 108" fill="none" stroke="#10233F" strokeWidth="4" strokeLinecap="round" />
          <circle cx="148" cy="110" r="4" fill="#39DDA1" />
        </g>

        <path className="avatar-signal" d="M43 36C55 22 72 17 90 17C108 17 125 22 137 36" fill="none" stroke="#38D99D" strokeWidth="4" strokeLinecap="round" opacity="0.85" />
      </svg>
    </div>
  );
}
