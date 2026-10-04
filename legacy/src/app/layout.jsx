import { Inter, Outfit, Geist } from "next/font/google";
import "./globals.css";

import { ToastContainer } from "react-toastify";
import "react-toastify/dist/ReactToastify.css";

import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import Provider from "@/components/SessionProvider";
import { ThemeProvider } from "@/components/ThemeProvider";
import GlobalVideoPlayer from "@/components/GlobalVideoPlayer";
import { cn } from "@/lib/utils";

const geist = Geist({subsets:['latin'],variable:'--font-sans'});
const outfit = Outfit({ subsets: ["latin"], variable: "--font-display" });
export const metadata = {
  title: "Aurahub",
  description: "A modern video streaming platform",
};

export default function RootLayout({ children }) {
  return (
    <html lang="en" className={cn("h-full", "font-sans", geist.variable)} suppressHydrationWarning>
      <body className={`${geist.variable} ${outfit.variable} font-sans flex flex-col h-full bg-background text-foreground transition-colors duration-300`}>
        <ThemeProvider attribute="class" defaultTheme="dark" enableSystem>
          <Provider>
            <ToastContainer
              position="top-right"
              autoClose={3000}
              hideProgressBar={false}
              newestOnTop={false}
              closeOnClick
              theme="light"
            />
            <Navbar />
            <div className="flex-grow">{children}</div>
            <Footer />
            <GlobalVideoPlayer />
          </Provider>
        </ThemeProvider>
      </body>
    </html>
  );
}
