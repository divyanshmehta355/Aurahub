import React from 'react';

const ContactPage = () => {
    return (
        <main className="container mx-auto px-6 py-12">
            <div className="max-w-3xl mx-auto text-center">
                <h1 className="text-4xl font-bold text-foreground mb-4">Contact Us</h1>
                <p className="text-lg text-muted-foreground mb-12">
                    We'd love to hear from you! Whether you have a question, feedback, or a concern, feel free to reach out.
                </p>
                
                <div className="bg-card p-8 md:p-10 rounded-2xl shadow-xl border border-border text-left transition-colors duration-300">
                    <h2 className="text-2xl font-bold text-foreground mb-6">Get in Touch</h2>
                    <div className="space-y-6">
                        <p className="text-muted-foreground leading-relaxed">
                            <strong className="text-foreground">Admin:</strong> For direct inquiries, you can contact the admin at <a href="mailto:kalluhalwai@aurahub.fun" className="font-semibold text-primary hover:text-primary transition-colors">divyansh@aurahub.fun</a>.
                        </p>
                        <p className="text-muted-foreground leading-relaxed">
                            <strong className="text-foreground">Support:</strong> For technical issues or help with your account, please email us at <a href="mailto:support@aurahub.fun" className="font-semibold text-primary hover:text-primary transition-colors">support@aurahub.fun</a>.
                        </p>
                        <p className="text-muted-foreground leading-relaxed">
                            <strong className="text-foreground">Report Issues:</strong> To report a bug or inappropriate content, contact us at <a href="mailto:report@aurahub.fun" className="font-semibold text-primary hover:text-primary transition-colors">report@aurahub.fun</a>.
                        </p>
                        <p className="text-muted-foreground leading-relaxed">
                            <strong className="text-foreground">Location:</strong> Aurahub HQ, Kota, Rajasthan, India.
                        </p>
                    </div>
                </div>
            </div>
        </main>
    );
};

export default ContactPage;