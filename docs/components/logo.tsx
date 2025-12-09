export function Logo() {
  return (
    <svg
      width="32"
      height="32"
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className="inline-block"
    >
      <rect width="32" height="32" rx="6" fill="currentColor" className="text-blue-600" />
      <path
        d="M8 24V8h4v12h8v4H8z"
        fill="white"
      />
      <path
        d="M22 8v8l-4 4v4l8-8V8h-4z"
        fill="white"
        opacity="0.8"
      />
    </svg>
  )
}
