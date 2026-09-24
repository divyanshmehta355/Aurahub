export const buildVideoAggregation = (
  filter = {},
  sortCriteria = { createdAt: -1 },
  pagination = null
) => {
  const isTrending = Boolean(sortCriteria && sortCriteria.trendingScore);

  const pipeline = [{ $match: filter }];

  if (isTrending) {
    // Trending calculation:
    pipeline.push(
      {
        $addFields: {
          likesCount: { $size: { $ifNull: ["$likes", []] } },
        },
      },
      {
        $lookup: {
          from: "comments",
          let: { videoId: "$_id" },
          pipeline: [
            { $match: { $expr: { $eq: ["$video", "$$videoId"] } } },
            { $project: { _id: 1 } },
          ],
          as: "comments",
        },
      },
      {
        $addFields: {
          commentCount: { $size: "$comments" },
          hoursSinceUpload: {
            $divide: [
              { $subtract: [new Date(), "$createdAt"] },
              3600000, // milliseconds in an hour
            ],
          },
        },
      },
      {
        $addFields: {
          trendingScore: {
            $divide: [
              {
                $add: [
                  "$views",
                  { $multiply: ["$likesCount", 5] },
                  { $multiply: ["$commentCount", 10] },
                ],
              },
              {
                $pow: [{ $add: ["$hoursSinceUpload", 2] }, 1.5],
              },
            ],
          },
        },
      },
      { $sort: sortCriteria }
    );

    if (pagination) {
      if (pagination.skip) pipeline.push({ $skip: pagination.skip });
      if (pagination.limit) pipeline.push({ $limit: pagination.limit });
    }
  } else {
    // Non-trending sorts: Sort using indexes first and paginate immediately
    pipeline.push({ $sort: sortCriteria });

    if (pagination) {
      if (pagination.skip) pipeline.push({ $skip: pagination.skip });
      if (pagination.limit) pipeline.push({ $limit: pagination.limit });
    }

    pipeline.push(
      {
        $addFields: {
          likesCount: { $size: { $ifNull: ["$likes", []] } },
        },
      },
      {
        $lookup: {
          from: "comments",
          let: { videoId: "$_id" },
          pipeline: [
            { $match: { $expr: { $eq: ["$video", "$$videoId"] } } },
            { $project: { _id: 1 } },
          ],
          as: "comments",
        },
      },
      {
        $addFields: {
          commentCount: { $size: "$comments" },
        },
      }
    );
  }

  // Join uploader details for the final paginated slice of videos
  pipeline.push(
    {
      $lookup: {
        from: "users",
        localField: "uploader",
        foreignField: "_id",
        as: "uploaderInfo",
      },
    },
    { $unwind: { path: "$uploaderInfo", preserveNullAndEmptyArrays: true } },
    {
      $project: {
        title: 1,
        description: 1,
        fileId: 1,
        thumbnailUrl: 1,
        category: 1,
        visibility: 1,
        views: 1,
        createdAt: 1,
        likesCount: 1,
        commentCount: 1,
        likes: 1,
        trendingScore: 1,
        "uploader.username": "$uploaderInfo.username",
        "uploader._id": "$uploaderInfo._id",
        "uploader.avatar": "$uploaderInfo.avatar",
      },
    }
  );

  return pipeline;
};
