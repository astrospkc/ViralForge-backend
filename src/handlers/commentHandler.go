package handlers

import (
	"fmt"
	"strconv"
	"viralforge/src/connect"
	"viralforge/src/models"

	"github.com/gofiber/fiber/v3"
	"github.com/uptrace/bun"
)

// create 1st comment
// reply comment
// delete comment
// edit comment
// like comments
// unlike comment

type CreateCommentResponse struct{
	Data 	models.Comment  
	Success bool 
	Code    int
	Message string 
}

func CreateComment() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, _ := FetchUserId(c)

		var req models.Comment
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(400).JSON(CreateCommentResponse{
				Success: false,
				Code:    400,
				Message: fmt.Sprintf("Error parsing body: %v", err),
			})
		}

		vID, _ := strconv.Atoi(c.Params("v_id"))
		videoID := int64(vID)

		var rootID int64
		var depth int = 0

		// 🧠 CASE 1: ROOT COMMENT
		if req.ParentCommentID == nil {
			rootID = 0 // temporary (will update later)
			depth = 0
		} else {
			// 🧠 CASE 2: REPLY
			parentID := *req.ParentCommentID

			parent := new(models.Comment)
			err := connect.Db.NewSelect().
				Model(parent).
				Where("id = ?", parentID).
				Scan(c.Context())

			if err != nil {
				return c.Status(400).JSON(CreateCommentResponse{
					Success: false,
					Code:    400,
					Message: "Parent comment not found",
				})
			}

			rootID = parent.RootCommentID
			depth = parent.Depth + 1
		}

		comment := &models.Comment{
			VideoID:         videoID,
			UserID:          userID,
			ParentCommentID: req.ParentCommentID,
			RootCommentID:   rootID, // 0 for root initially
			Content:         req.Content,
			Rating:          req.Rating,
			LikeCount:       req.LikeCount,
			ReplyCount:      req.ReplyCount,
			Depth:           depth,
		}

		// 🔥 INSERT
		err := connect.Db.NewInsert().
			Model(comment).
			Returning("*").
			Scan(c.Context())

		if err != nil {
			return c.Status(400).JSON(CreateCommentResponse{
				Success: false,
				Code:    400,
				Message: fmt.Sprintf("Insert error: %v", err),
			})
		}

		// 🔥 ONLY for root comment → update root_id
		if req.ParentCommentID == nil {
			_, err = connect.Db.NewUpdate().
				Model((*models.Comment)(nil)).
				Where("id = ?", comment.ID).
				Set("root_comment_id = ?", comment.ID).
				Exec(c.Context())

			if err != nil {
				return c.Status(400).JSON(CreateCommentResponse{
					Success: false,
					Code:    400,
					Message: fmt.Sprintf("Update root error: %v", err),
				})
			}

			comment.RootCommentID = comment.ID
		}

		return c.Status(200).JSON(CreateCommentResponse{
			Data:    *comment,
			Success: true,
			Code:    200,
			Message: "Comment created successfully",
		})
	}
}

type UpdateCommentResponse struct {
	Success bool 
	Code    int 
	Message string
}

func UpdateComment() fiber.Handler{
	return func (c fiber.Ctx) error{
		comment_id, _ := strconv.Atoi(c.Params("comment_id"))
		c_id := int64(comment_id)



		var body  struct{
			Content  string `json:"string"`

		}

		if err:= c.Bind().Body(&body); err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(UpdateCommentResponse{
				Success: false,
				Code :400,
				Message: fmt.Sprintf("Error while fetching data from request body : ", err),
			})
		}

		err := connect.Db.NewUpdate().Model((*models.Comment)(nil)).Set("content = ?", body.Content).Where("id = ?", c_id).Returning("*").Scan(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(UpdateCommentResponse{
				Code:400,
				Message: fmt.Sprintf("Error while editing comment: ", err),
				Success: false,
			})
		}

		return c.Status(fiber.StatusAccepted).JSON(UpdateCommentResponse{
			Code:200,
			Message:"Successfully edited comment",
			Success: true,
		})
		
	}
}

func DeleteComment() fiber.Handler{
	return func (c fiber.Ctx) error{
		comment_id,_:=strconv.Atoi(c.Params("comment_id"))

		_,err := connect.Db.NewUpdate().Model((*models.Comment)(nil)).Set("status = ?", models.CommentHidden).Where("id = ?",comment_id).Exec(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"Message":fmt.Sprintf("Error while deleting comment: ", err),
				"Code":400,
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message":fmt.Sprintf(" Successfully deleted comment"),
			"Code":200,
		})
		
	}
}

type GetCommentsResponse struct{
	Data	[]models.Comment 
	Success bool 
	Code    int 
	Messaage string
}



// BUT WHEN SENDING COMMENTS ARRAY , WE NEED TO SEND THE COMMENTS AS WELL AS USER INFO: JOIN THE USER AND COMMENTS TABLE TO FETCH THE INFO.

// this is the top level comments that's been sent to the frontend , where we need video id only and no parent_id 
func GetTopLevelComments() fiber.Handler{
	return func (c fiber.Ctx) error{
		video_id, _ := strconv.Atoi(c.Params("v_id"))
		v_id:= int64(video_id)

		var comments []models.Comment 

		err := connect.Db.NewSelect().Model(&comments).Relation("User", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Column("id", "name", "email", "created_at", "updated_at")
		}).Where("video_id = ?", v_id).Where("parent_comment_id IS NULL").Where("status = ?", models.CommentVisible).Order("created_at ASC").Scan(c.Context())
		if err!=nil{
			fmt.Printf("error: ", err)
			return c.Status(fiber.StatusBadRequest).JSON(GetCommentsResponse{
				Success:false,
				Code: 400,
				Messaage: "Failed to get comments",
			})
		}

		return c.Status(fiber.StatusOK).JSON(GetCommentsResponse{
			Data:    comments,
			Success: true,
			Code:200,
			Messaage: "Successfully fetched all the comments",
		})
		
	}
}



func GetReplies() fiber.Handler{
	return func(c fiber.Ctx) error {
		// get parent comment id
		comment_id, err := strconv.Atoi(c.Params("parent_comment_id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(GetCommentsResponse{
				Success:  false,
				Code:     400,
				Messaage: "Invalid comment ID",
			})
		}
		c_id := int64(comment_id)

		var replies []models.Comment
		// Also fetch the user associated with each reply
		err = connect.Db.NewSelect().
			Model(&replies).
			Relation("User", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Column("id", "name", "email", "created_at", "updated_at")
			}).
			Where("parent_comment_id = ?", c_id).
			Where("status = ?", models.CommentVisible). // Only get visible comments
			Order("created_at ASC").                   // Show oldest replies first
			Scan(c.Context())

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(GetCommentsResponse{
				Success:  false,
				Code:     500,
				Messaage: fmt.Sprintf("Failed to fetch replies: %v", err),
			})
		}

		return c.Status(fiber.StatusOK).JSON(GetCommentsResponse{
			Data:     replies,
			Success:  true,
			Code:     200,
			Messaage: "Successfully fetched all replies",
		})
	}
}

// type Reviews struct{
// 	id  int64;
//     userId int64;
//     userName string;
//     avatar  string;
//     rating  int16;
//     comment []string;
// }

// func GetReviews() fiber.Handler{
// 	return func(c fiber.Ctx) error{
// 		var reviews []Reviews 
// 		video_id, _ := strconv.Atoi(c.Params("v_id"))
// 		v_id:= int64(video_id)
		



// 		var comments []models.Comment
// 		err := connect.Db.NewSelect().Model(&comments).Where("video_id = ?", v_id).Scan(c.Context())
// 		if err!=nil{
// 			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 				"message":"failed to fetch reviews",
// 				"code":400,
// 			})
// 		}
// 		// user info:
// 		var user_info models.User
// 		err = connect.Db.NewSelect().Model(&user_info).Where("id = ?", user_id).Scan(c.Context())
// 		if err!=nil{
// 			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 				"message":"failed to fetch reviews",
// 				"code":400,
// 			})
// 		}

// 		reviews = append(reviews,)
		


		

// 	}
// }