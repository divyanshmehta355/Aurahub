import React from 'react';
import Link from 'next/link';

const Footer = () => {
    return (
        <footer className="bg-muted/30 border-t border-border mt-12 transition-colors duration-300">
            <div className="container mx-auto px-6 py-8 text-center text-muted-foreground">
                <div className="flex justify-center space-x-6 mb-4">
                    <Link href="/about" className="text-sm hover:text-foreground transition-colors">About</Link>
                    <Link href="/contact" className="text-sm hover:text-foreground transition-colors">Contact</Link>
                    <Link href="/terms" className="text-sm hover:text-foreground transition-colors">Terms of Service</Link>
                    <Link href="/privacy" className="text-sm hover:text-foreground transition-colors">Privacy Policy</Link>
                </div>
                <p className="text-sm">&copy; {new Date().getFullYear()} Aurahub. All Rights Reserved.</p>
            </div>
        </footer>
    );
};

export default Footer;