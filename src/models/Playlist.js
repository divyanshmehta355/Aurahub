import mongoose from 'mongoose';

const playlistSchema = new mongoose.Schema({
    title: {
        type: String,
        required: true,
        trim: true,
    },
    description: {
        type: String,
        trim: true,
    },
    owner: {
        type: mongoose.Schema.Types.ObjectId,
        ref: 'User',
        required: true,
    },
    videos: [{
        type: mongoose.Schema.Types.ObjectId,
        ref: 'Video'
    }],
    isPublic: {
        type: Boolean,
        default: true,
    },
}, { timestamps: true });

playlistSchema.index({ owner: 1, createdAt: -1 });
playlistSchema.index({ isPublic: 1, createdAt: -1 });

const Playlist = mongoose.models.Playlist || mongoose.model('Playlist', playlistSchema);

export default Playlist;