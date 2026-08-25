import { SignIn } from "@clerk/nextjs";

export const metadata = { title: "Sign in" };

export default function SignInPage() {
  return (
    <div className="flex justify-center py-20 px-5">
      <SignIn />
    </div>
  );
}
