import mongoose from 'mongoose';
import CATEGORIES from '@/constants/categories';

const videoSchema = new mongoose.Schema({
    title: {
        type: String,
        required: true,
        trim: true
    },
    description: {
        type: String,
        required: false,
        default: ""
    },
    fileId: {
        type: String,
        required: true,
        unique: true
    },
    thumbnailUrl: {
        type: String,
        required: false
    },
    category: {
        type: String,
        enum: CATEGORIES,
        default: "Other",
    },
    tags: {
        type: [String],
        default: [],
    },
    visibility: {
        type: String,
        enum: ['public', 'unlisted', 'private'],
        default: 'public',
    },
    uploader: {
        type: mongoose.Schema.Types.ObjectId,
        ref: 'User',
        required: true,
    },
    isShort: {
        type: Boolean,
        default: false
    },
    views: {
        type: Number,
        default: 0
    },
    likes: [{
        type: mongoose.Schema.Types.ObjectId,
        ref: 'User'
    }],
    embedding: {
        type: [Number],
        default: undefined,
        select: false,
    },
    streamtapeUrl: {
        type: String,
        required: false,
    },
    lastRefreshedAt: {
        type: Date,
        default: Date.now,
    },
    pendingRemoteUploadId: {
        type: String,
        default: null,
    },
    streamtapeStatus: {
        type: String,
        enum: ['active', 'dead', 'pending'],
        default: 'active',
    },
    cloneAttempts: {
        type: Number,
        default: 0,
    },
}, { timestamps: true });

videoSchema.index({
    title: 'text',
    description: 'text',
    category: 'text',
    tags: 'text'
});

videoSchema.index({ uploader: 1, createdAt: -1 });
videoSchema.index({ category: 1, views: -1 });
videoSchema.index({ visibility: 1 });
videoSchema.index({ visibility: 1, isShort: 1, createdAt: -1 });
videoSchema.index({ visibility: 1, isShort: 1, views: -1 });
videoSchema.index({ visibility: 1, category: 1, isShort: 1, createdAt: -1 });
videoSchema.index({ visibility: 1, category: 1, isShort: 1, views: -1 });
videoSchema.index({ streamtapeStatus: 1, lastRefreshedAt: 1, createdAt: 1 });
videoSchema.index({ pendingRemoteUploadId: 1 }, { sparse: true });


const Video = mongoose.models.Video || mongoose.model('Video', videoSchema);

export default Video;