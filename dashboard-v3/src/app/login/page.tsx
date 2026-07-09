"use client";

import Image from "next/image";
import { Suspense } from "react";
import { LoginForm } from "./login-form";

export default function LoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="w-full max-w-sm space-y-6 px-4">
        <div className="space-y-2 text-center">
          <Image
            src="/logo.svg"
            alt="Castellan"
            width={48}
            height={84}
            className="mx-auto"
          />
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            Castellan
          </h1>
          <p className="text-sm text-muted-foreground">
            Sign in to your dashboard
          </p>
        </div>
        <Suspense
          fallback={
            <div className="flex justify-center py-8">
              <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
            </div>
          }
        >
          <LoginForm />
        </Suspense>
      </div>
    </div>
  );
}
