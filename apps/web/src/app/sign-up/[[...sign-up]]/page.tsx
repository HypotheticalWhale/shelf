import { SignUp } from "@clerk/nextjs";

export const metadata = { title: "Join" };

export default function SignUpPage() {
  return (
    <div className="flex justify-center py-20 px-5">
      <SignUp />
    </div>
  );
}
