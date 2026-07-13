import { useEffect, useRef } from "react";

interface FormErrorProps {
  id?: string;
  message: string;
}

export function FormError({ id, message }: FormErrorProps) {
  const ref = useRef<HTMLParagraphElement>(null);

  useEffect(() => {
    if (message) ref.current?.focus();
  }, [message]);

  if (!message) return null;

  return (
    <p
      ref={ref}
      id={id}
      role="alert"
      tabIndex={-1}
      className="text-sm text-destructive"
    >
      {message}
    </p>
  );
}
